// examples/servo_sweep.go
// Smooth servo sweep using the Servo library.

package main

import (
	"arduino"
	"Servo"
)

const servoPin = 9
const minAngle = 0
const maxAngle = 180
const stepMs   = 15

var myServo Servo.Servo
var angle    int32
var forward  bool

func setup() {
	Servo.Attach(myServo, servoPin)
	angle   = 0
	forward = true
}

func loop() {
	Servo.Write(myServo, angle)
	arduino.delay(stepMs)

	if forward {
		angle++
		if angle >= maxAngle {
			forward = false
		}
	} else {
		angle--
		if angle <= minAngle {
			forward = true
		}
	}
}