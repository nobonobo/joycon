package joycon

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	GyroRange = 16000
	SensorRes = 65535
	GyroGain  = 4000
	//AccelK     = GyroRange / SensorRes / 1000
	AccelK = 1.0 / 4096
	//GyroK      = GyroGain / SensorRes
	GyroK = 1.0 / 4096
)

// Stick ...
type Stick struct {
	X int16
	Y int16
}

// String ...
func (s Stick) String() string {
	return fmt.Sprintf("[%d %d]", s.X, s.Y)
}

// Vec2 ...
type Vec2 struct {
	X float32
	Y float32
}

func (v Vec2) String() string {
	return fmt.Sprintf(
		"[%4.1f %4.1f]",
		v.X, v.Y,
	)
}

// Vec3 ...
type Vec3 struct {
	X float32
	Y float32
	Z float32
}

func (v Vec3) String() string {
	return fmt.Sprintf(
		"[%4.1f %4.1f %4.1f]",
		v.X, v.Y, v.Z,
	)
}

// State ...
type State struct {
	Tick     byte
	Battery  int
	Buttons  uint32
	Left     Stick
	Right    Stick
	LeftAdj  Vec2
	RightAdj Vec2
	Err      error
}

// UnmarshalBinary ...
func (s *State) UnmarshalBinary(b []byte) error {
	if len(b) < 15 {
		return fmt.Errorf("invalid bytes length")
	}
	s.Tick = b[1]
	s.Battery = (int(b[2]) >> 4) * 100 / 8
	s.Buttons = uint32(b[3]) | uint32(b[4])<<8 | uint32(b[5])<<16
	if !bytes.Equal(b[6:9], []byte{0, 0, 0}) {
		s.Left.X = int16(b[6]) | int16(b[7]&0x0f)<<8
		s.Left.Y = int16(b[7]>>4) | int16(b[8])<<4
	}
	if !bytes.Equal(b[9:12], []byte{0, 0, 0}) {
		s.Right.X = int16(b[9]) | int16(b[10]&0x0f)<<8
		s.Right.Y = int16(b[10]>>4) | int16(b[11])<<4
	}
	return nil
}

// Sensor ...
type Sensor struct {
	Tick  byte
	Gyro  Vec3
	Accel Vec3
}

// SensorTri ...
type SensorTri [3]Sensor

// UnmarshalBinary ...
func (s *SensorTri) UnmarshalBinary(b []byte) error {
	if len(b) < 49 {
		return fmt.Errorf("invalid bytes length")
	}
	for n := 0; n < 3; n++ {
		s[n].Tick = b[1] - byte(2-n)
		s[n].Accel.X = AccelK * float32(int16(binary.LittleEndian.Uint16(b[13+n*12:15+n*12])))
		s[n].Accel.Y = AccelK * float32(int16(binary.LittleEndian.Uint16(b[15+n*12:17+n*12])))
		s[n].Accel.Z = AccelK * float32(int16(binary.LittleEndian.Uint16(b[17+n*12:19+n*12])))
		s[n].Gyro.X = GyroK * float32(int16(binary.LittleEndian.Uint16(b[19+n*12:21+n*12])))
		s[n].Gyro.Y = GyroK * float32(int16(binary.LittleEndian.Uint16(b[21+n*12:23+n*12])))
		s[n].Gyro.Z = GyroK * float32(int16(binary.LittleEndian.Uint16(b[23+n*12:25+n*12])))
	}
	return nil
}

// CalibInfo ...
type CalibInfo struct {
	Center Stick
	Min    Stick
	Max    Stick
}

// UnmarshalBinary ...
func (ci *CalibInfo) UnmarshalBinary(b []byte) error {
	ci.Max.X = int16(b[0]) | int16(b[1]&0xf)<<8
	ci.Max.Y = int16(b[1]>>4) | int16(b[2])<<4
	ci.Center.X = int16(b[3]) | int16(b[4]&0xf)<<8
	ci.Center.Y = int16(b[4]>>4) | int16(b[5])<<4
	ci.Min.X = int16(b[6]) | int16(b[7]&0xf)<<8
	ci.Min.Y = int16(b[7]>>4) | int16(b[8])<<4
	return nil
}

/*
func (ci *CalibInfo) String() string {
	max := Stick{ci.Center.X - ci.Max.X, ci.Center.Y - ci.Max.Y}
	min := Stick{ci.Center.X - ci.Min.X, ci.Center.Y - ci.Min.Y}
	return fmt.Sprintf("{%v %v %v}", ci.Center, max, min)
}
*/
