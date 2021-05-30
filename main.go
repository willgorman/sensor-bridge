package main

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/vrecan/death/v3"
)

type Sensor struct {
	sync.RWMutex
	Name        string
	Addr        string
	Parser      AdvParser
	Updates     <-chan ble.Advertisement
	currReading *Reading
	updtTime    time.Time
	// TODO: expire the reading if it has been too long since an update
}

func (s *Sensor) Run(ctx context.Context) {
	log.Infof("starting sensor %s", s.Name)
	go func() {
		defer log.Infof("halting sensor %s", s.Name)
		for {
			select {
			case <-ctx.Done():
				return
			case adv := <-s.Updates:
				if adv == nil {
					// update channel closed, we're done
					return
				}
				r, err := s.Parser.Parse(adv)

				if err != nil {
					log.Printf("parsing advertisement for %s: %s", s.Name, err)
					continue
				}
				s.Lock()
				s.currReading = r
				s.updtTime = time.Now()
				s.Unlock()
				log.Info(s.String())
			}
		}

	}()
}

func (s *Sensor) String() string {
	s.RLock()
	defer s.RUnlock()
	if s.currReading == nil {
		return fmt.Sprintf("[%s] - no reading", s.Name)
	}
	return fmt.Sprintf("[%s] %s (as of %s)", s.Name, s.currReading, s.updtTime.Format(time.RFC3339))
}

func (s *Sensor) IsStale() bool {
	if s.currReading == nil {
		return true
	}
	// TODO: add threshold config for updtTime too old
	return false
}

func (s *Sensor) Temperature() float64 {
	s.RLock()
	defer s.RUnlock()
	if s.currReading == nil {
		return math.NaN()
	}
	return float64(s.currReading.Temp)
}

func (s *Sensor) Humidity() float64 {
	s.RLock()
	defer s.RUnlock()
	if s.currReading == nil {
		return math.NaN()
	}
	return float64(s.currReading.Humidity)
}

func (s *Sensor) Battery() float64 {
	s.RLock()
	defer s.RUnlock()
	if s.currReading == nil {
		return math.NaN()
	}
	return float64(s.currReading.Battery)
}

type Reading struct {
	Temp     float32
	Humidity float32
	Battery  int8
}

func (r Reading) String() string {
	return fmt.Sprintf("%.2f ℃  | %.2f%% humidity | %d%% battery", r.Temp, r.Humidity, r.Battery)
}

type ScannerOptions struct {
	Duration time.Duration
	Interval time.Duration
}

func startScanner(ctx context.Context, advRouter map[string]chan<- ble.Advertisement, options ScannerOptions) {
	log.Infof("Scanner options: %#v", options)
	go func() {
		fmt.Printf("Scanning for %s...\n", options.Duration)
		blectx := ble.WithSigHandler(context.WithTimeout(context.Background(), options.Duration))
		chkErr(ble.Scan(blectx, true, handler(advRouter), supportedDeviceFilter))
		interval := time.NewTicker(options.Interval)
		for {
			select {
			case <-ctx.Done():
			case <-interval.C:
				// Scan for specified durantion, or until interrupted by user.
				log.Infof("Scanning for %s...\n", options.Duration)
				blectx := ble.WithSigHandler(context.WithTimeout(context.Background(), options.Duration))
				chkErr(ble.Scan(blectx, true, handler(advRouter), supportedDeviceFilter))
			}
		}
	}()
}

func handler(advRouter map[string]chan<- ble.Advertisement) func(a ble.Advertisement) {
	return func(a ble.Advertisement) {
		upC, ok := advRouter[a.Addr().String()]
		if !ok {
			// not a sensor we're tracking
			log.Infof("ignoring advertisement for %s", a.Addr())
			return
		}
		upC <- a
	}
}

func chkErr(err error) {
	switch errors.Cause(err) {
	case nil:
	case context.DeadlineExceeded:
		log.Infof("scanning done\n")
	case context.Canceled:
		log.Infof("scanning canceled\n")
	default:
		// the scanner should keep running so just log the unexpected error
		log.Errorf("scanning: %s", err.Error())
	}
}

func init() {
	viper.AutomaticEnv()
	viper.SetDefault("Scanner.Duration", 15*time.Second)
	viper.SetDefault("Scanner.Interval", 5*time.Minute)
	viper.SetDefault("prometheus.port", 2112)
	viper.AddConfigPath("/etc/sensor-bridge")
	viper.AddConfigPath("$HOME/.sensorbridge")
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	// viper.SetConfigType("toml")

}

func main() {
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	d, err := dev.NewDevice("default")
	if err != nil {
		log.Fatalf("can't create ble device: %s", err)
	}
	ble.SetDefaultDevice(d)

	var sensors []*Sensor
	err = viper.UnmarshalKey("sensors", &sensors)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startScanner(ctx, initSensors(ctx, sensors), ScannerOptions{
		Duration: viper.GetDuration("Scanner.Duration"),
		Interval: viper.GetDuration("Scanner.Interval"),
	})

	for _, s := range sensors {
		CreateGauges(s)
	}

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(fmt.Sprintf(":%d", viper.GetInt("prometheus.port")), nil)

	go func() {
		for {
			tick := time.NewTicker(10 * time.Second)
			select {
			case <-tick.C:
				for _, s := range sensors {
					log.Println(s.String())
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	// block until done
	death := death.NewDeath(syscall.SIGINT, syscall.SIGTERM)
	death.WaitForDeath()
}

func supportedDeviceFilter(a ble.Advertisement) bool {
	supported := strings.HasPrefix(a.LocalName(), "GVH5102")
	if supported {
		log.Infof("allow %s: %t", a.LocalName(), supported)
	}

	return supported
}

func initSensors(ctx context.Context, s []*Sensor) map[string]chan<- ble.Advertisement {
	m := map[string]chan<- ble.Advertisement{}
	for _, i := range s {
		c := make(chan ble.Advertisement)
		i.Updates = c
		m[i.Addr] = c
		i.Run(ctx)
		i.Parser = H5102Parser{}
	}
	return m
}
