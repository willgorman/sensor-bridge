package main

import (
	"math"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/pkg/value"
)

func CreateGauges(s *Sensor) {
	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace:   "sensor",
		Subsystem:   "govee",
		Name:        "temperature",
		ConstLabels: prometheus.Labels{"name": s.Name},
	}, func() float64 {
		if s.IsStale() {
			return math.Float64frombits(value.StaleNaN)
		}
		return s.Temperature()
	})
	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace:   "sensor",
		Subsystem:   "govee",
		Name:        "humidity",
		ConstLabels: prometheus.Labels{"name": s.Name},
	}, func() float64 {
		if s.IsStale() {
			return math.Float64frombits(value.StaleNaN)
		}
		return s.Humidity()
	})
	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace:   "sensor",
		Subsystem:   "govee",
		Name:        "battery_level",
		ConstLabels: prometheus.Labels{"name": s.Name},
	}, func() float64 {
		if s.IsStale() {
			return math.Float64frombits(value.StaleNaN)
		}
		return s.Battery()
	})
}
