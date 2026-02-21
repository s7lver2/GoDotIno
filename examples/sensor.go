// examples/sensor.go
// Reads a temperature sensor on A0 and prints values over Serial.

package main

import (
	"arduino"
	"fmt"
	"math"
)

const sensorPin = arduino.A0
const vRef      = 5.0
const numSamples = 10

var baseline float32

func setup() {
	arduino.Serial.Begin(9600)
	fmt.Println("Sensor init OK")
	baseline = readVoltage()
}

func loop() {
	var total float64
	for i := 0; i < numSamples; i++ {
		raw := arduino.analogRead(sensorPin)
		total += float64(raw)
		arduino.delay(10)
	}

	avg    := total / float64(numSamples)
	volts  := (avg / 1023.0) * vRef
	celsius := (volts - 0.5) * 100.0
	celsius  = math.Round(celsius*100) / 100

	fmt.Println(celsius)
	arduino.delay(1000)
}

func readVoltage() float32 {
	raw := arduino.analogRead(sensorPin)
	return float32(raw) / 1023.0 * vRef
}