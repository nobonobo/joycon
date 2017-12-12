# joycon

Nintendo Switch's Joycon Device access library(via bluetooth only)

## Reverse engineering info

https://github.com/dekuNukem/Nintendo_Switch_Reverse_Engineering

## Feature

- supported deveces: Joycon(L/R), Pro-Controller
- get: Digial Buttons state
- get: Analog Sticks state
- set: Raw Vibration data
- calibration support for analog stick.

## Dependencies

- go get -u github.com/flynn/hid
- go get -u github.com/shibukawa/gotomation `optional`

## Usage

```go
package main

import "github.com/nobonobo/joycon"

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
    fmt.Println(s.Buttons)  // Button bits
    fmt.Println(s.LeftAdj)  // Left Analog Stick State
    fmt.Println(s.RightAdj) // Right Analog Stick State
    a := <-jc.Sensor()
    fmt.Println(a.Accel) // Acceleration Sensor State
    fmt.Println(a.Gyro)  // Gyro Sensor State

    jc.Close()
}
```
## TODO

- [ ] Deadzone parameter read from SPI memory. 
- [ ] Rich Vibration support.
- [ ] Set Player LED.
- [ ] Set HomeButton LED.
- [ ] Low power mode support.
- [ ] IR sensor capture.
