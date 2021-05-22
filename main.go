package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type Address string

type Sensor struct {
	Name string
	Addr Address
}

type Reading struct {
	Temp     float32
	Humidity float32
	Battery  int8
}

type ScannerOptions struct {
	Duration time.Duration
	Interval time.Duration
}

func startScanner(ctx context.Context, sensors map[Address]chan<- Reading, options ScannerOptions) {

	go func() {
		interval := time.NewTicker(options.Interval)
		select {
		case <-ctx.Done():
		case <-interval.C:
			// Scan for specified durantion, or until interrupted by user.
			fmt.Printf("Scanning for %s...\n", options.Duration)
			blectx := ble.WithSigHandler(context.WithTimeout(context.Background(), options.Duration))
			chkErr(ble.Scan(blectx, true, nil, supportedDeviceFilter))
		}
	}()
}

func chkErr(err error) {
	switch errors.Cause(err) {
	case nil:
	case context.DeadlineExceeded:
		fmt.Printf("done\n")
	case context.Canceled:
		fmt.Printf("canceled\n")
	default:
		// the scanner should keep running so just log the unexpected error
		log.Print(err.Error())
	}
}

func init() {
	viper.AutomaticEnv()
	viper.SetDefault("ScannerInterval", 5*time.Minute)
	viper.SetDefault("ScannerDuration", 15*time.Second)
	viper.AddConfigPath("$HOME/.sensorbridge")
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	// viper.SetConfigType("toml")

}

func main() {
	d, err := dev.NewDevice("default")
	if err != nil {
		log.Fatalf("can't create ble device: %s", err)
	}
	ble.SetDefaultDevice(d)

}

func supportedDeviceFilter(a ble.Advertisement) bool {
	return strings.HasPrefix(a.LocalName(), "GVH5102")
}
