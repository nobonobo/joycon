package joycon

// Ref: https://github.com/dekuNukem/Nintendo_Switch_Reverse_Engineering/blob/bd12f564a9281ba61ab7b7782dc0255c642cb5e4/bluetooth_hid_subcommands_notes.md

import (
	"bytes"
	"fmt"
	"math"
	"sync"

	"github.com/flynn/hid"
)

var (
	connectSeq = [][]byte{
		{0x30, 0x01}, // Set PlayerLED
		{0x01, 0x01}, // Connect1
		{0x01, 0x02}, // Connect2
		{0x01, 0x03}, // Connect3
		{0x40, 0x01}, // Enable 6axis Sensor
		{0x03, 0x30}, // Set Standard full mode. Pushes current state @60Hz
		{0x48, 0x01}, // Enable Vibration
	}
	disconnectSeq = [][]byte{
		{0x30, 0x00},
		{0x40, 0x00},
		{0x48, 0x00},
		{0x03, 0x3f},
		{0x01, 0x06},
	}
)

// Joycon ...
type Joycon struct {
	closeOnce       sync.Once
	device          hid.Device
	rumble, command chan []byte
	report          chan []byte
	state           chan State
	sensor          chan Sensor
	count           byte
	leftEnable      bool
	rightEnable     bool
	leftStick       CalibInfo
	rightStick      CalibInfo
}

// NewJoycon ...
func NewJoycon(devicePath string) (*Joycon, error) {
	jc := &Joycon{
		rumble:  make(chan []byte, 16),
		command: make(chan []byte, 16),
		report:  make(chan []byte, 16),
		state:   make(chan State, 16),
		sensor:  make(chan Sensor, 16),
	}
	info, err := hid.ByPath(devicePath)
	if err != nil {
		return nil, err
	}
	device, err := info.Open()
	if err != nil {
		return nil, err
	}
	jc.device = device
	go jc.receive()
	go jc.run()
	return jc, nil
}

// Close ...
func (jc *Joycon) Close() {
	jc.closeOnce.Do(func() {
		close(jc.rumble)
		close(jc.command)
		for range jc.state {
		}
		jc.device.Close()
	})
}

// State ...
func (jc *Joycon) State() <-chan State {
	return jc.state
}

// Sensor ...
func (jc *Joycon) Sensor() <-chan Sensor {
	return jc.sensor
}

// Rumble ...
func (jc *Joycon) Rumble(b []byte) {
	for len(b) >= 8 {
		jc.rumble <- b[:8]
		b = b[8:]
	}
	if len(b) > 0 {
		jc.rumble <- b
	}
}

// LeftStickCalibration ...
func (jc *Joycon) LeftStickCalibration() CalibInfo {
	return jc.leftStick
}

// RightStickCalibration ...
func (jc *Joycon) RightStickCalibration() CalibInfo {
	return jc.rightStick
}

func (jc *Joycon) subcommand(rumble, cmd []byte) ([]byte, error) {
	r := []byte{0x00, 0x01, 0x40, 0x40, 0x00, 0x01, 0x40, 0x40}
	copy(r, rumble)
	buf := make([]byte, 0, 24)
	buf = append(buf, []byte{0x01, jc.count}...)
	buf = append(buf, r...)
	buf = append(buf, cmd...)
	jc.count = (jc.count + 1) & 15
	if err := jc.device.Write(buf); err != nil {
		return nil, err
	}
	rep, ok := <-jc.report
	if !ok {
		return nil, jc.device.ReadError()
	}
	return rep, nil
}

// ReadSPI ...
func (jc *Joycon) ReadSPI(addr uint16, length int) ([]byte, error) {
	var rep []byte
	for i := 0; i < 100; i++ {
		r, err := jc.subcommand(nil, []byte{0x10, byte(addr & 0xff), byte(addr >> 8), 0x00, 0x00, byte(length)})
		if err != nil {
			return nil, err
		}
		if uint16(r[15])|uint16(r[16])<<8 == addr {
			rep = r[20 : 20+length]
			break
		}
	}
	return rep, nil
}

func (jc *Joycon) receive() {
	defer close(jc.report)
	for {
		rep, ok := <-jc.device.ReadCh()
		if !ok {
			return
		}
		switch rep[0] {
		case 0x30: // gyro & accel
			s := SensorTri{}
			if err := s.UnmarshalBinary(rep); err != nil {
				return
			}
			for n := 0; n < 3; n++ {
				select {
				case jc.sensor <- s[n]:
				default:
				}
			}
		default:
			jc.report <- rep
		}
	}
}

func (jc *Joycon) run() {
	defer close(jc.state)
	if _, err := jc.subcommand(nil, []byte{0x03, 0x3f}); err != nil {
		jc.state <- State{Err: err}
		return
	}
	data, err := jc.ReadSPI(0x6012, 1)
	if err != nil {
		jc.state <- State{Err: err}
		return
	}
	switch data[0] {
	case 0x01:
		jc.leftEnable = true
	case 0x02:
		jc.rightEnable = true
	case 0x03:
		jc.leftEnable = true
		jc.rightEnable = true
	default:
		jc.state <- State{Err: fmt.Errorf("unknown product type: %d", data[0])}
		return
	}
	if jc.leftEnable {
		data, err = jc.ReadSPI(0x8012, 9)
		if err != nil {
			jc.state <- State{Err: err}
			return
		}
		if !bytes.Equal(data, bytes.Repeat([]byte{0xff}, 9)) {
			jc.leftStick.UnmarshalBinary(data)
		} else {
			data, err = jc.ReadSPI(0x603d, 9)
			if err != nil {
				jc.state <- State{Err: err}
				return
			}
			if !bytes.Equal(data, bytes.Repeat([]byte{0xff}, 9)) {
				jc.leftStick.UnmarshalBinary(data)

			}
		}
	}
	if jc.rightEnable {
		data, err = jc.ReadSPI(0x801d, 9)
		if err != nil {
			jc.state <- State{Err: err}
			return
		}
		if !bytes.Equal(data, bytes.Repeat([]byte{0xff}, 9)) {
			d := make([]byte, 0, 9)
			d = append(d, data[6:9]...)
			d = append(d, data[0:3]...)
			d = append(d, data[3:6]...)
			jc.rightStick.UnmarshalBinary(d)
		} else {
			data, err = jc.ReadSPI(0x6046, 9)
			if err != nil {
				jc.state <- State{Err: err}
				return
			}
			if !bytes.Equal(data, bytes.Repeat([]byte{0xff}, 9)) {
				d := make([]byte, 0, 9)
				d = append(d, data[6:9]...)
				d = append(d, data[0:3]...)
				d = append(d, data[3:6]...)
				jc.rightStick.UnmarshalBinary(d)
			}
		}
	}
	// LeftStick Deadzone
	data, err = jc.ReadSPI(0x6086, 16)
	if err != nil {
		jc.state <- State{Err: err}
		return
	}
	// RightStick Deadzone
	data, err = jc.ReadSPI(0x6098, 16)
	if err != nil {
		jc.state <- State{Err: err}
		return
	}
	// Gyro Parameters (user)
	data, err = jc.ReadSPI(0x8034, 10)
	if err != nil {
		jc.state <- State{Err: err}
		return
	}
	// Gyro Parameters (sys)
	data, err = jc.ReadSPI(0x6029, 10)
	if err != nil {
		jc.state <- State{Err: err}
		return
	}
	for _, seq := range connectSeq {
		_, err := jc.subcommand(nil, seq)
		if err != nil {
			jc.state <- State{Err: err}
			return
		}
	}
	defer func() {
		for _, seq := range disconnectSeq {
			_, err := jc.subcommand(nil, seq)
			if err != nil {
				jc.state <- State{Err: err}
				return
			}
		}
	}()
	// loop
	for {
		var r, c []byte
		select {
		case v, ok := <-jc.command:
			if !ok {
				return
			}
			c = v
		default:
		}
		select {
		case v, ok := <-jc.rumble:
			if !ok {
				return
			}
			r = v
		default:
		}
		rep, err := jc.subcommand(r, c)
		if err != nil {
			jc.state <- State{Err: err}
			return
		}
		switch rep[0] {
		case 0x3f:
		case 0x31, 0x32, 0x33:
		case 0x21:
			s := &State{}
			if err := s.UnmarshalBinary(rep); err != nil {
				jc.state <- State{Err: err}
				return
			}
			if jc.leftEnable {
				s.LeftAdj = jc.calibration(jc.leftStick, s.Left)
			}
			if jc.rightEnable {
				s.RightAdj = jc.calibration(jc.rightStick, s.Right)
			}
			select {
			case jc.state <- *s:
			default:
			}
		}
	}
}

func (jc *Joycon) calibration(c CalibInfo, s Stick) Vec2 {
	var res Vec2
	diff := float32(s.X) - float32(c.Center.X)
	if math.Abs(float64(diff)) < 0xae { // TODO: deadzone from SPI
		diff = 0.0
	}
	if diff > 0 {
		res.X = diff / float32(c.Max.X)
	} else {
		res.X = diff / float32(c.Min.X)
	}
	diff = float32(s.Y) - float32(c.Center.Y)
	if diff > 0 {
		res.Y = diff / float32(c.Max.Y)
	} else {
		res.Y = diff / float32(c.Min.Y)
	}
	return res
}
