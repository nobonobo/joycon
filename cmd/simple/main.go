package main

import (
	"fmt"
	"log"

	"github.com/nobonobo/joycon"
)

func main() {
	devices, err := joycon.Search()
	if err != nil {
		log.Fatalln(err)
	}
	if len(devices) == 0 {
		log.Fatalln("joycon not found")
	}
	jc, err := joycon.NewJoycon(devices[0].Path)
	if err != nil {
		log.Fatalln(err)
	}
	s := <-jc.State()
	fmt.Printf("%#v\n", s.Buttons)  // Button bits
	fmt.Printf("%#v\n", s.LeftAdj)  // Left Analog Stick State
	fmt.Printf("%#v\n", s.RightAdj) // Right Analog Stick State
	a := <-jc.Sensor()
	fmt.Printf("%#v\n", a.Accel) // Acceleration Sensor State
	fmt.Printf("%#v\n", a.Gyro)  // Gyro Sensor State

	jc.Close()
}
