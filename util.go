package joycon

import (
	"fmt"

	"github.com/flynn/hid"
)

type DeviceType int

const (
	JoyConL DeviceType = 0x2006
	JoyConR DeviceType = 0x2007
	ProCon  DeviceType = 0x2009
)

// Search ...
func Search(dts ...DeviceType) ([]*hid.DeviceInfo, error) {
	res := []*hid.DeviceInfo{}
	devices, err := hid.Devices()
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		if device.VendorID != 0x057e {
			continue
		}
		switch device.ProductID {
		default:
			continue
		case 0x2006, 0x2007, 0x2009:
			if len(dts) == 0 {
				res = append(res, device)
				continue
			}
			dt := DeviceType(device.ProductID)
			for _, t := range dts {
				if dt == t {
					res = append(res, device)
				}
			}
		}
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("not found device")
	}
	return res, nil
}
