package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/nobonobo/joycon"
	"github.com/shibukawa/gotomation"
)

var (
	mouse      gotomation.Mouse
	keyboard   gotomation.Keyboard
	oldButtons uint32
	oldStick   joycon.Vec2
	oldBattery int
	rumbleData = []joycon.RumbleSet{
		{
			{HiFreq: 64, HiAmp: 80, LoFreq: 64, LoAmp: 80}, // HiCoil
			{HiFreq: 64, HiAmp: 80, LoFreq: 64, LoAmp: 80}, // LoCoil
		},
		{
			{HiFreq: 64, HiAmp: 80, LoFreq: 64, LoAmp: 80}, // HiCoil
			{HiFreq: 64, HiAmp: 80, LoFreq: 64, LoAmp: 80}, // LoCoil
		},
		{
			{HiFreq: 64, HiAmp: 0, LoFreq: 64, LoAmp: 0}, // HiCoil
			{HiFreq: 64, HiAmp: 0, LoFreq: 64, LoAmp: 0}, // LoCoil
		},
	}
)

func init() {
	screen, err := gotomation.GetMainScreen()
	if err != nil {
		log.Fatalln(err)
	}
	mouse = screen.Mouse()
	keyboard = screen.Keyboard()
}

// Joycon ...
type Joycon struct {
	dx, dy int
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
		keyboard.KeyDown(gotomation.VK_SHIFT)
	case downButtons>>7&1 == 1: // ZR
		keyboard.KeyDown(gotomation.VK_CONTROL)
	case downButtons>>0&1 == 1: // Y
		jc.SendRumble(rumbleData...)
		mouse.ClickWith(gotomation.MouseLeft)
	case downButtons>>1&1 == 1: // X
		jc.SendRumble(rumbleData...)
		mouse.ClickWith(gotomation.MouseCenter)
	case downButtons>>2&1 == 1: // B
		jc.SendRumble(rumbleData...)
		mouse.ClickWith(gotomation.MouseRight)
	case downButtons>>3&1 == 1: // A
		keyboard.KeyDown(gotomation.VK_SPACE)
	case downButtons>>4&1 == 1: // SR
		mouse.Scroll(+2, 0, 30*time.Millisecond)
	case downButtons>>5&1 == 1: // SL
		mouse.Scroll(-2, 0, 30*time.Millisecond)
	case downButtons>>9&1 == 1: // +
		keyboard.KeyDown(gotomation.VK_ESCAPE)
	case downButtons>>10&1 == 1: // RStick Push
	case downButtons>>12&1 == 1: // Home
	}
	switch {
	case upButtons == 0:
	default:
		log.Printf("up  : %06X", upButtons)
	case upButtons>>6&1 == 1: // R
		keyboard.KeyUp(gotomation.VK_SHIFT)
	case upButtons>>7&1 == 1: // ZR
		keyboard.KeyUp(gotomation.VK_CONTROL)
	case upButtons>>0&1 == 1: // Y
	case upButtons>>1&1 == 1: // X
	case upButtons>>2&1 == 1: // B
	case upButtons>>3&1 == 1: // A
		keyboard.KeyUp(gotomation.VK_SPACE)
	case upButtons>>4&1 == 1: // SR
	case upButtons>>5&1 == 1: // SL
	case upButtons>>9&1 == 1: // +
		keyboard.KeyUp(gotomation.VK_ESCAPE)
	case upButtons>>10&1 == 1: // RStick Push
	case upButtons>>12&1 == 1: // Home
	}
	switch {
	case s.RightAdj.X > 0.5 && oldStick.X < 0.5:
		keyboard.KeyDown(gotomation.VK_RIGHT)
	case s.RightAdj.X < 0.5 && oldStick.X > 0.5:
		keyboard.KeyUp(gotomation.VK_RIGHT)
	}
	switch {
	case s.RightAdj.X < -0.5 && oldStick.X > -0.5:
		keyboard.KeyDown(gotomation.VK_LEFT)
	case s.RightAdj.X > -0.5 && oldStick.X < -0.5:
		keyboard.KeyUp(gotomation.VK_LEFT)
	}
	switch {
	case s.RightAdj.Y > 0.5 && oldStick.Y < 0.5:
		keyboard.KeyDown(gotomation.VK_UP)
	case s.RightAdj.Y < 0.5 && oldStick.Y > 0.5:
		keyboard.KeyUp(gotomation.VK_UP)
	}
	switch {
	case s.RightAdj.Y < -0.5 && oldStick.Y > -0.5:
		keyboard.KeyDown(gotomation.VK_DOWN)
	case s.RightAdj.Y > -0.5 && oldStick.Y < -0.5:
		keyboard.KeyUp(gotomation.VK_DOWN)
	}
}

func (jc *Joycon) apply() {
	if jc.dx != 0 || jc.dy != 0 {
		x, y := mouse.GetPosition()
		mouse.MoveQuickly(x+jc.dx, y+jc.dy)
		jc.dx = 0
		jc.dy = 0
	}
}

func (jc *Joycon) sensorHandle(s joycon.Sensor) {
	if jc.IsLeft() || jc.IsProCon() {
		jc.dx -= int(s.Gyro.Z * 200)
		jc.dy += int(s.Gyro.Y * 200)
	}
	if jc.IsRight() {
		jc.dx += int(s.Gyro.Z * 200)
		jc.dy -= int(s.Gyro.Y * 200)
	}
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	devices, err := joycon.Search(joycon.JoyConR)
	if err != nil {
		log.Fatalln(err)
	}
	j, err := joycon.NewJoycon(devices[0].Path)
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
