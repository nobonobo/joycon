package joycon

import "github.com/flynn/hid"

// Search ...
func Search() ([]*hid.DeviceInfo, error) {
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
		}
		res = append(res, device)
	}
	return res, nil
}
