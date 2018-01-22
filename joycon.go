package joycon

// Ref: https://github.com/dekuNukem/Nintendo_Switch_Reverse_Engineering/blob/bd12f564a9281ba61ab7b7782dc0255c642cb5e4/bluetooth_hid_subcommands_notes.md

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flynn/hid"
)

var (
	connectSeq = [][]byte{
		/*
			{0x01, 0x01}, // Connect1
			{0x01, 0x02}, // Connect2
			{0x01, 0x03}, // Connect3
		*/
		{0x30, 0x01}, // Set PlayerLED
		{0x40, 0x01}, // Enable 6axis Sensor
		{0x48, 0x01}, // Enable Vibration
		{0x03, 0x30}, // Set Standard full mode. Pushes current state @60Hz
	}
	disconnectSeq = [][]byte{
		{0x30, 0x00}, // Clear PlayerLED
		{0x40, 0x00}, // Disable 6axis Sensor
		{0x48, 0x00}, // Disable Vibration
		{0x03, 0x3f}, // Set Normal HID Mode
		//{0x06, 0x00}, // HCI Disconnect
	}
)

type sub struct {
	cmd []byte
	rep chan<- []byte
}

// Joycon ...
type Joycon struct {
	info         *hid.DeviceInfo
	closeOnce    sync.Once
	device       hid.Device
	rumble       chan []byte
	report       chan []byte
	state        chan State
	sensor       chan Sensor
	irdata       chan IRData
	irenable     bool
	outputcode   byte
	sub          chan sub
	closing      chan struct{}
	done         chan struct{}
	color        color.Color
	count        byte
	leftEnable   bool
	rightEnable  bool
	leftStick    CalibInfo
	rightStick   CalibInfo
	stats        Stats
	sendRumble   chan<- []byte
	muSendRumble sync.RWMutex
	interval     *time.Ticker
}

// NewJoycon ...
func NewJoycon(devicePath string, irenable bool) (*Joycon, error) {
	jc := &Joycon{
		rumble:     make(chan []byte, 6),
		report:     make(chan []byte, 16),
		state:      make(chan State, 16),
		sensor:     make(chan Sensor, 16),
		irdata:     make(chan IRData, 16),
		irenable:   irenable,
		outputcode: 0x01,
		sub:        make(chan sub),
		closing:    make(chan struct{}),
		done:       make(chan struct{}),
		interval:   time.NewTicker(5 * time.Millisecond),
	}
	jc.sendRumble = jc.rumble
	info, err := hid.ByPath(devicePath)
	if err != nil {
		return nil, err
	}
	jc.info = info
	device, err := info.Open()
	if err != nil {
		return nil, err
	}
	jc.device = device
	go jc.receive()
	if err := jc.subcommand(nil, []byte{0x03, 0x3f}); err != nil {
		return nil, err
	}
	if _, err := jc.reply(); err != nil {
		return nil, err
	}
	data, err := jc.ReadSPI(0x6012, 1)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("unknown product type: %d", data[0])
	}
	go jc.run()
	return jc, nil
}

// Close ...
func (jc *Joycon) Close() {
	jc.closeOnce.Do(func() {
		jc.muSendRumble.Lock()
		close(jc.closing)
		jc.sendRumble = nil
		jc.muSendRumble.Unlock()
		close(jc.rumble)
		<-jc.done
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

// IRData ...
func (jc *Joycon) IRData() <-chan IRData {
	return jc.irdata
}

// Subcommand ...
func (jc *Joycon) Subcommand(b []byte) ([]byte, error) {
	ch := make(chan []byte, 1)
	jc.sub <- sub{
		cmd: b,
		rep: ch,
	}
	rep, ok := <-ch
	if !ok {
		return nil, fmt.Errorf("reply receive failed")
	}
	return rep, nil
}

// SendRumble ...
func (jc *Joycon) SendRumble(rs ...RumbleSet) error {
	for _, r := range rs {
		b, err := r.MarshalBinary()
		if err != nil {
			return err
		}
		jc.muSendRumble.RLock()
		select {
		case <-jc.closing:
			return io.EOF
		case jc.sendRumble <- b:
		}
		jc.muSendRumble.RUnlock()
	}
	return nil
}

// IsLeft ...
func (jc *Joycon) IsLeft() bool {
	return jc.leftEnable && !jc.rightEnable
}

// IsRight ...
func (jc *Joycon) IsRight() bool {
	return !jc.leftEnable && jc.rightEnable
}

// IsProCon ...
func (jc *Joycon) IsProCon() bool {
	return jc.leftEnable && jc.rightEnable
}

// LeftStickCalibration ...
func (jc *Joycon) LeftStickCalibration() CalibInfo {
	return jc.leftStick
}

// RightStickCalibration ...
func (jc *Joycon) RightStickCalibration() CalibInfo {
	return jc.rightStick
}

// Name ...
func (jc *Joycon) Name() string {
	return jc.info.Product
}

// Stats ...
func (jc *Joycon) Stats() Stats {
	return Stats{
		RumbleCount: atomic.LoadUint64(&jc.stats.RumbleCount),
		SensorCount: atomic.LoadUint64(&jc.stats.SensorCount),
		IRDataCount: atomic.LoadUint64(&jc.stats.IRDataCount),
		StateCount:  atomic.LoadUint64(&jc.stats.StateCount),
	}
}

func (jc *Joycon) subcommand(rumble, cmd []byte) error {
	defer func() { <-jc.interval.C }()
	buf := make([]byte, 0x40)
	if len(cmd) == 0 {
		buf[0] = 0x10
	} else {
		buf[0] = jc.outputcode
	}
	buf[1] = jc.count
	copy(buf[2:10], rumble)
	copy(buf[10:], cmd)
	jc.count = (jc.count + 1) & 15
	return jc.device.Write(buf)
}

func (jc *Joycon) reply() ([]byte, error) {
	rep, ok := <-jc.report
	if !ok {
		return nil, fmt.Errorf("report closed")
	}
	return rep, nil
}

// ReadSPI ...
func (jc *Joycon) ReadSPI(addr uint16, length int) ([]byte, error) {
	var rep []byte
	for i := 0; i < 100; i++ {
		if err := jc.subcommand(nil, []byte{0x10, byte(addr & 0xff), byte(addr >> 8), 0x00, 0x00, byte(length)}); err != nil {
			return nil, err
		}
		r, err := jc.reply()
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
		select {
		case rep, ok := <-jc.device.ReadCh():
			if !ok {
				return
			}
			switch rep[0] {
			case 0x30:
				// gyro & accel
				s := Sensors{}
				if err := s.UnmarshalBinary(rep); err != nil {
					return
				}
				atomic.AddUint64(&jc.stats.SensorCount, 1)
				for n := 0; n < 3; n++ {
					select {
					case jc.sensor <- s[n]:
					default:
					}
				}
				continue
			case 0x31:
				// IR data
				data := IRData{}
				if err := data.UnmarshalBinary(rep[49:362]); err != nil {
					log.Println(err)
					return
				}
				atomic.AddUint64(&jc.stats.IRDataCount, 1)
				select {
				case jc.irdata <- data:
				default:
				}
				continue
			case 0x3f:
				log.Println("")
			case 0x32, 0x33:
				log.Printf("rep: %X", rep)
			case 0x21:
				s := &State{}
				if err := s.UnmarshalBinary(rep); err == nil {
					if jc.leftEnable {
						s.LeftAdj = jc.calibration(jc.leftStick, s.Left)
					}
					if jc.rightEnable {
						s.RightAdj = jc.calibration(jc.rightStick, s.Right)
					}
				} else {
					s.Err = err
				}
				select {
				case jc.state <- *s:
					atomic.AddUint64(&jc.stats.StateCount, 1)
				default:
				}
			default:
			}
			jc.report <- rep
		}
	}
}

func (jc *Joycon) run() {
	defer close(jc.done)
	if jc.leftEnable {
		data, err := jc.ReadSPI(0x8012, 9)
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
	data, err := jc.ReadSPI(0x6050, 3)
	if err != nil {
		jc.state <- State{Err: err}
		return
	}
	jc.color = color.RGBA{data[0], data[1], data[2], 0}
	if jc.rightEnable {
		data, err := jc.ReadSPI(0x801d, 9)
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
	_, err = jc.ReadSPI(0x6086, 16)
	if err != nil {
		jc.state <- State{Err: err}
		return
	}
	// RightStick Deadzone
	_, err = jc.ReadSPI(0x6098, 16)
	if err != nil {
		jc.state <- State{Err: err}
		return
	}
	// Gyro Parameters (user)
	_, err = jc.ReadSPI(0x8034, 10)
	if err != nil {
		jc.state <- State{Err: err}
		return
	}
	// Gyro Parameters (sys)
	_, err = jc.ReadSPI(0x6029, 10)
	if err != nil {
		jc.state <- State{Err: err}
		return
	}
	for _, seq := range connectSeq {
		if err := jc.subcommand(nil, seq); err != nil {
			jc.state <- State{Err: err}
			return
		}
		if _, err := jc.reply(); err != nil {
			jc.state <- State{Err: err}
			return
		}
	}
	defer func() {
		for _, seq := range disconnectSeq {
			if err := jc.subcommand(nil, seq); err != nil {
				jc.state <- State{Err: err}
				return
			}
			if _, err := jc.reply(); err != nil {
				jc.state <- State{Err: err}
				return
			}
		}
	}()
	if jc.irenable {
		if err := jc.subcommand(nil, []byte{0x40, 0x00}); err != nil {
			jc.state <- State{Err: err}
			return
		}
		r, err := jc.reply()
		if err != nil {
			log.Println(err)
			jc.state <- State{Err: err}
			return
		}
		log.Printf("r:%X", r)
		if err := jc.subcommand(nil, []byte{0x03, 0x31}); err != nil {
			jc.state <- State{Err: err}
			return
		}
		r, err = jc.reply()
		if err != nil {
			log.Println(err)
			jc.state <- State{Err: err}
			return
		}
		log.Printf("r:%X", r)
		if err := jc.subcommand(nil, []byte{0x11, 0x03, 0x00}); err != nil {
			log.Println(err)
			jc.state <- State{Err: err}
			return
		}
		r, err = jc.reply()
		if err != nil {
			log.Println(err)
			jc.state <- State{Err: err}
			return
		}
		log.Printf("r:%X", r)
		jc.outputcode = 0x11
		if err := jc.subcommand(nil, []byte{0x03, 0x00}); err != nil {
			log.Println(err)
			jc.state <- State{Err: err}
			return
		}
		/*
			r, err = jc.reply()
			if err != nil {
				log.Println(err)
				jc.state <- State{Err: err}
				return
			}
			log.Printf("r:%X", r)
		*/
		jc.outputcode = 0x1
	}
	// loop
	t := time.NewTicker(time.Millisecond * 15)
	t2 := time.NewTimer(time.Millisecond * 120)
	r := []byte{0x00, 0x01, 0x40, 0x40, 0x00, 0x01, 0x40, 0x40}
	for {
		select {
		case v := <-jc.sub:
			if err := jc.subcommand(r, v.cmd); err != nil {
				v.rep <- nil
			}
			b, err := jc.reply()
			if err != nil {
				v.rep <- nil
			}
			v.rep <- b
		case v, ok := <-jc.rumble:
			if !ok {
				return
			}
			r = v
			atomic.AddUint64(&jc.stats.RumbleCount, 1)
			if err := jc.subcommand(r, nil); err != nil {
				log.Println(err)
				jc.state <- State{Err: err}
				return
			}
			t2.Reset(time.Millisecond * 120)
		case <-t2.C:
			r = []byte{0x00, 0x01, 0x40, 0x40, 0x00, 0x01, 0x40, 0x40}
		case <-t.C:
			select {
			default:
			case v, ok := <-jc.rumble:
				if !ok {
					return
				}
				r = v
			}
			if err := jc.subcommand(r, []byte{0}); err != nil {
				log.Println(err)
				jc.state <- State{Err: err}
				return
			}
			jc.reply()
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
