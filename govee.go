package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-ble/ble"
)

type AdvParser interface {
	Parse(a ble.Advertisement) (*Reading, error)
}

type H5102Parser struct{}

func (p H5102Parser) Parse(a ble.Advertisement) (*Reading, error) {
	if !strings.HasPrefix(a.LocalName(), "GVH5102") {
		return &Reading{}, errors.New("advertisement from wrong device type")
	}

	battery := int(a.ManufacturerData()[7])
	temp, humidity, err := decodeReading(a)
	if err != nil {
		return nil, err
	}

	return &Reading{
		Battery:  int8(battery),
		Temp:     temp,
		Humidity: humidity,
	}, nil
}

func decodeReading(a ble.Advertisement) (float32, float32, error) {
	data := a.ManufacturerData()[4:7]
	hex := fmt.Sprintf("%X", data)
	val, err := strconv.ParseInt(hex, 16, 64)
	if err != nil {
		return 0.0, 0.0, fmt.Errorf("parsing payload: %s", err)
	}
	humidity := float32((val % 1000) / 10) // TODO: doesn't seem to match iOS reading
	if (val & 0x800000) != 0 {
		return float32((val ^ 0x800000) / -10000), humidity, nil
	}

	return float32(val / 10000), humidity, nil
}
