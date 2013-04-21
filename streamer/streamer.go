package main

import (
	"flag"
	"fmt"
	"github.com/jbrukh/goavatar"
	"github.com/jbrukh/gplot"
	"github.com/jbrukh/window"
	"log"
)

const DEFAULT_PORT = "/dev/tty.AvatarEEG03009-SPPDev"

var serialPort *string = flag.String("port", DEFAULT_PORT, "the serial port for the device")

func init() {
	flag.Parse()
}

func main() {
	const WINDOW = 10000
	// set up the plotter	
	p, err := gplot.NewPlotter(false)
	if err != nil {
		err_string := fmt.Sprintf("** err: %v\n", err)
		panic(err_string)
	}
	defer p.Close()

	//p.CheckedCmd("set yrange [0.01:0.018]")
	p.CheckedCmd(fmt.Sprintf("set xrange [0:%v]", WINDOW))
	p.CheckedCmd("set terminal aqua size 1430,400")

	// set up the device
	device := goavatar.NewDevice(DEFAULT_PORT)
	out, err := device.Connect()
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	window1, window2 := window.New(WINDOW, 10), window.New(WINDOW, 10)

	for i := 0; i < 10000; i++ {
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
		if i%5 == 0 {
			p.Dual(window1.Slice(), window2.Slice(), "Ch1", "Ch2")
		}
	}
	log.Printf("Finished... closing.")
	device.Disconnect()
}
