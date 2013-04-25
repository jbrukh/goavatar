package main

import (
	"github.com/jbrukh/goavatar/socket"
	"net/http"
)

const (
	DefaultPort        = "/dev/tty.AvatarEEG03009-SPPDev"
	DefaultWindowSize  = 10000
	DefaultRefreshRate = 5
	DefaultMaxFrames   = 10000
	WindowMultiple     = 10
)

var (
	serialPort  *string = flag.String("port", DefaultPort, "the serial port for the device")
	refreshRate *int    = flag.Int("refreshRate", DefaultRefreshRate, "the number of data points to buffer before refreshing")
	maxFrames   *int    = flag.Int("maxFrames", DefaultMaxFrames, "maximum frames to read before turning off")
	mockDevice  *bool   = flag.Bool("mockDevice", false, "whether to use the mock device")
)

func main() {

}
