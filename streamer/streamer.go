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
	// set up the plotter	
	p, err := gplot.NewPlotter(false)
	if err != nil {
		err_string := fmt.Sprintf("** err: %v\n", err)
		panic(err_string)
	}
	defer p.Close()

	//p.CheckedCmd("set yrange [0:0.5]")
	p.CheckedCmd(fmt.Sprintf("set xrange [0:%v]", 1000))
	p.CheckedCmd("set terminal aqua size 1430,400")

	// set up the device
	device := goavatar.NewDevice(DEFAULT_PORT)
	out, err := device.Connect()
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	window := window.New(1000, 10)

	for i := 0; i < 1000; i++ {
		df, ok := <-out
		if !ok {
			log.Printf("The data channel got closed (exiting)")
			return
		}
		//log.Printf("Got df: %v", df.String())
		for _, v := range df.ChannelData(1) {
			window.PushBack(v)
		}

		// now display it
		log.Printf("slice: %v", window.Slice())
		p.PlotX(window.Slice(), "AvatarEEG")
	}
	log.Printf("Finished... closing.")
	device.Disconnect()
}
