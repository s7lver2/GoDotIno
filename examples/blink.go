// examples/blink.go
// Classic blink — the Arduino "Hello World".
// Transpile with:  goduino examples/blink.go build/blink.cpp --board uno

package main

import (
	"arduino"
	"fmt"
)

const ledPin = 13
const blinkInterval = 500 // milliseconds

func setup() {
	arduino.pinMode(ledPin, arduino.OUTPUT)
	fmt.Println("Blink ready!")
}

func loop() {
	arduino.digitalWrite(ledPin, arduino.HIGH)
	arduino.delay(blinkInterval)
	arduino.digitalWrite(ledPin, arduino.LOW)
	arduino.delay(blinkInterval)
}