package devices

import (
	"flag"
	"fmt"
)

var (
	repo         *string = flag.String("repo", DefaultRepo, "directory where recordings are stored")
	serialPort   *string = flag.String("serialPort", DefaultSerialPort, "the serial port for the device")
	mockDevice   *bool   = flag.Bool("mockDevice", false, "whether to use the mock device")
	mockFile     *string = flag.String("mockFile", DefaultMockFile, "OBF file to play back in the mock device")
	mockChannels *int    = flag.Int("mockChannels", DefaultMockChannels, "the number of channels to mock in the mock device")
)

var deviceMap map[string]Device

func init() {
	flag.Parse()
	deviceMap = map[string]Device{
		"avatar":      NewAvatarDevice(*repo, *serialPort),
		"mock_avatar": NewMockDevice(*repo, *mockFile, *mockChannels),
	}
}

// Provides a new instance of a supported
// device. Options:
//
//    "avatar"
//    "mock_avatar"
//
func Provide(device string) Device {
	dev, ok := deviceMap[device]
	if ok {
		return dev
	} else {
		msg := fmt.Sprintf("unknown device: %s", device)
		panic(msg)
	}
}
