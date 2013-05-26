//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package drivers

import (
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar/device"
	. "github.com/jbrukh/goavatar/drivers/avatar"
	. "github.com/jbrukh/goavatar/drivers/mock_avatar"
	. "github.com/jbrukh/goavatar/drivers/thinkgear"
)

const (
	DefaultSerialPort   = "/dev/tty.AvatarEEG04024-SPPDev"
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

// Provides a new instance of a supported
// device. Options:
//
//    "avatar"
//    "mock_avatar"
//    "thinkgear"
//
// Must be called after initialize().
func provide(device string) (Device, error) {
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
	initialize()
	if *mockDevice {
		return provide("mock_avatar")
	}
	return provide(*device)
}

func initialize() {
	if !flag.Parsed() {
		flag.Parse()
	}
	if deviceMap == nil {
		deviceMap = map[string]Device{
			"avatar":      NewAvatarDevice(*repo, *port),
			"mock_avatar": NewMockDevice(*repo, *mockFile, *mockChannels),
			"thinkgear":   NewThinkGearDevice(*repo, *port),
		}
	}
}
