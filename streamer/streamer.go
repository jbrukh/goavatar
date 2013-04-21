package main

import (
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar"
	"github.com/jbrukh/gplot"
	"github.com/jbrukh/window"
	"log"
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
	windowSize  *int    = flag.Int("windowSize", DefaultWindowSize, "the number of data points in the viewing window")
	refreshRate *int    = flag.Int("refreshRate", DefaultRefreshRate, "the number of data points to buffer before refreshing")
	maxFrames   *int    = flag.Int("maxFrames", DefaultMaxFrames, "maximum frames to read before turning off")
	mockDevice  *bool   = flag.Bool("mockDevice", false, "whether to use the mock device")
)

func init() {
	flag.Parse()
}

func main() {
	// set up the plotter	
	p, err := gplot.NewPlotter(false)
	if err != nil {
		err_string := fmt.Sprintf("** err: %v\n", err)
		panic(err_string)
	}
	defer p.Close()

	//p.CheckedCmd("set yrange [0.01:0.018]")
	p.CheckedCmd(fmt.Sprintf("set xrange [0:%v]", *windowSize))
	p.CheckedCmd("set terminal aqua size 1430,400")

	// set up the device
	var device Device
	if *mockDevice {
		device = NewMockDevice()
	} else {
		device = NewAvatarDevice(*serialPort)
	}

	// connect to it
	out, err := device.Connect()
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	run(p, out)

	log.Printf("Thank you!")
	device.Disconnect()
}

func run(p *gplot.Plotter, out <-chan *DataFrame) {
	var (
		window1 = window.New(*windowSize, WindowMultiple)
		window2 = window.New(*windowSize, WindowMultiple)
	)
	for i := 0; i < *maxFrames; i++ {
		df, ok := <-out
		if !ok {
			log.Printf("The data channel got closed (exiting)")
			return
		}
		//log.Printf("Got df: %v", df.String())
		for _, v := range df.ChannelData(1) {
			window1.PushBack(v)
		}
		for _, v := range df.ChannelData(2) {
			window2.PushBack(v)
		}

		// now display it
		//log.Printf("slice: %v", window.Slice())
		//p.PlotX(window.Slice(), "AvatarEEG")
		if i%*refreshRate == 0 {
			p.Dual(window1.Slice(), window2.Slice(), "Ch1", "Ch2")
		}
	}
}
