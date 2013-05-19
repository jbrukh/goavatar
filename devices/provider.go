//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package devices

import (
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar"
	. "github.com/jbrukh/goavatar/devices/avatar"
	. "github.com/jbrukh/goavatar/devices/mock_avatar"
	. "github.com/jbrukh/goavatar/devices/thinkgear"
)

const (
	DefaultSerialPort   = "/dev/tty.AvatarEEG03009-SPPDev"
	DefaultRepo         = "var"
	DefaultMockFile     = "etc/1fabece1-7a57-96ab-3de9-71da8446c52c"
	DefaultMockChannels = 4
	DefaultDevice       = "avatar"
)

var (
	repo         *string = flag.String("repo", DefaultRepo, "directory where recordings are stored")
	port         *string = flag.String("port", DefaultSerialPort, "the serial port for the device")
	mockDevice   *bool   = flag.Bool("mockDevice", false, "whether to use the mock device")
	mockFile     *string = flag.String("mockFile", DefaultMockFile, "OBF file to play back in the mock device")
	mockChannels *int    = flag.Int("mockChannels", DefaultMockChannels, "the number of channels to mock in the mock device")
	device       *string = flag.String("device", DefaultDevice, "one of {'avatar', 'mock_avatar', 'thinkgear'}")
)

// devices
var deviceMap map[string]Device

func init() {
	flag.Parse()

	deviceMap = map[string]Device{
		"avatar":      NewAvatarDevice(*repo, *port),
		"mock_avatar": NewMockDevice(*repo, *mockFile, *mockChannels),
		"thinkgear":   NewThinkGearDevice(*repo, *port),
	}
}

// Provides a new instance of a supported
// device. Options:
//
//    "avatar"
//    "mock_avatar"
//    "thinkgear"
//
func Provide(device string) (Device, error) {
	dev, ok := deviceMap[device]
	if ok {
		return dev, nil
	} else {
		return nil, fmt.Errorf("unknown device: %s", device)
	}
}

// Convenience function for working with the
// command line.
func ProvideDevice() (Device, error) {
	if *mockDevice {
		return Provide("mock_avatar")
	}
	return Provide(*device)
}
