package main

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/viper"
)

func TestConfig(t *testing.T) {
	viper.ReadInConfig()
	spew.Dump(viper.ConfigFileUsed())
	spew.Dump(viper.GetDuration("scannerinterval"))
	spew.Dump(viper.GetStringMap("scanner"))
	var sensors []Sensor
	viper.UnmarshalKey("sensors", &sensors)

	var opts ScannerOptions
	viper.UnmarshalKey("scanner", &opts)
	spew.Dump(sensors)
	spew.Dump(opts)
	spew.Dump(viper.GetDuration("Scanner.Interval"))
}
