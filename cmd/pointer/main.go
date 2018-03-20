package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/go-vgo/robotgo"
	"github.com/nobonobo/joycon"
)

var (
	oldButtons uint32
	oldStick   joycon.Vec2
	oldBattery int
	rumbleData = []joycon.RumbleSet{
		{
			{HiFreq: 64, HiAmp: 0, LoFreq: 64, LoAmp: 0},   // HiCoil
			{HiFreq: 16, HiAmp: 80, LoFreq: 16, LoAmp: 80}, // LoCoil
		},
		{
			{HiFreq: 64, HiAmp: 0, LoFreq: 64, LoAmp: 0}, // HiCoil
			{HiFreq: 64, HiAmp: 0, LoFreq: 64, LoAmp: 0}, // LoCoil
		},
	}
)

// Joycon ...
type Joycon struct {
	dx, dy    float32
	stop      bool
	scroll    bool
	scrollPos float32
	*joycon.Joycon
}

func (jc *Joycon) stateHandle(s joycon.State) {
	defer func() {
		oldButtons = s.Buttons
		oldStick = s.RightAdj
	}()
	if oldBattery != s.Battery {
		log.Println("battery:", s.Battery, "%")
	}
	oldBattery = s.Battery
	downButtons := s.Buttons & ^oldButtons
	upButtons := ^s.Buttons & oldButtons
	switch {
	case downButtons == 0:
	default:
		log.Printf("down: %06X", downButtons)
	case downButtons>>6&1 == 1: // R
		jc.stop = true
	case downButtons>>7&1 == 1: // ZR
		jc.scroll = true
	case downButtons>>0&1 == 1: // Y
		jc.SendRumble(rumbleData...)
		robotgo.MouseClick("left")
	case downButtons>>1&1 == 1: // X
		jc.SendRumble(rumbleData...)
		robotgo.MouseClick("center")
	case downButtons>>3&1 == 1: // A
		jc.SendRumble(rumbleData...)
		robotgo.MouseClick("right")
	case downButtons>>2&1 == 1: // B
		robotgo.KeyTap("space")
	case downButtons>>4&1 == 1: // SR
		robotgo.Scroll(0, -2)
	case downButtons>>5&1 == 1: // SL
		robotgo.Scroll(0, +2)
	case downButtons>>9&1 == 1: // +
		robotgo.KeyTap("escape")
	case downButtons>>10&1 == 1: // RStick Push
	case downButtons>>12&1 == 1: // Home
	}
	switch {
	case upButtons == 0:
	default:
		log.Printf("up  : %06X", upButtons)
	case upButtons>>6&1 == 1: // R
		jc.stop = false
	case upButtons>>7&1 == 1: // ZR
		jc.scroll = false
	case upButtons>>0&1 == 1: // Y
	case upButtons>>1&1 == 1: // X
	case upButtons>>2&1 == 1: // B
	case upButtons>>3&1 == 1: // A
	case upButtons>>4&1 == 1: // SR
	case upButtons>>5&1 == 1: // SL
	case upButtons>>9&1 == 1: // +
	case upButtons>>10&1 == 1: // RStick Push
	case upButtons>>12&1 == 1: // Home
	}
	if jc.scroll {
		jc.scrollPos += s.RightAdj.Y * s.RightAdj.Y * s.RightAdj.Y
		d := -int(jc.scrollPos)
		jc.scrollPos += float32(d)
		robotgo.Scroll(0, d)
	} else {
		switch {
		case s.RightAdj.X > 0.5 && oldStick.X < 0.5:
			robotgo.KeyTap("right")
		case s.RightAdj.X < 0.5 && oldStick.X > 0.5:
		}
		switch {
		case s.RightAdj.X < -0.5 && oldStick.X > -0.5:
			robotgo.KeyTap("left")
		case s.RightAdj.X > -0.5 && oldStick.X < -0.5:
		}
		switch {
		case s.RightAdj.Y > 0.5 && oldStick.Y < 0.5:
			robotgo.KeyTap("up")
		case s.RightAdj.Y < 0.5 && oldStick.Y > 0.5:
		}
		switch {
		case s.RightAdj.Y < -0.5 && oldStick.Y > -0.5:
			robotgo.KeyTap("down")
		case s.RightAdj.Y > -0.5 && oldStick.Y < -0.5:
		}
	}
}

func (jc *Joycon) apply() {
	if (jc.dx != 0 || jc.dy != 0) && !jc.stop {
		x, y := robotgo.GetMousePos()
		w, h := robotgo.GetScreenSize()
		x += int(jc.dx)
		y += int(jc.dy)
		if x >= w {
			x = w
		}
		if x < 0 {
			x = 0
		}
		if y >= h {
			y = h
		}
		if y < 0 {
			y = 0
		}
		robotgo.MoveMouse(x, y)
		jc.dx = 0
		jc.dy = 0
	}
}

func (jc *Joycon) sensorHandle(s joycon.Sensor) {
	if jc.IsLeft() || jc.IsProCon() {
		jc.dx -= s.Gyro.Z * 64
		jc.dy += s.Gyro.Y * 64
	}
	if jc.IsRight() {
		jc.dx += s.Gyro.Z * 64
		jc.dy -= s.Gyro.Y * 64
	}
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	devices, err := joycon.Search(joycon.JoyConR)
	if err != nil {
		log.Fatalln(err)
	}
	j, err := joycon.NewJoycon(devices[0].Path, false)
	if err != nil {
		log.Fatalln(err)
	}
	defer j.Close()
	jc := &Joycon{Joycon: j}
	log.Println("connected:", jc.Name())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	for {
		select {
		case <-sig:
			return
		case s, ok := <-jc.State():
			if !ok {
				return
			}
			jc.stateHandle(s)
			jc.apply()
		case s, ok := <-jc.Sensor():
			if !ok {
				return
			}
			jc.sensorHandle(s)
		}
	}
}
