package main

import (
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar"
	"github.com/jbrukh/gplot"
	"github.com/jbrukh/window"
	"log"
	"os"
	"text/template"
)

const (
	DefaultPort        = "/dev/tty.AvatarEEG03009-SPPDev"
	DefaultWindowSize  = 10000
	DefaultRefreshRate = 5
	DefaultMaxFrames   = 10000
	WindowMultiple     = 10
	DumpFile           = "frames.go"
)

var (
	serialPort  *string = flag.String("port", DefaultPort, "the serial port for the device")
	windowSize  *int    = flag.Int("windowSize", DefaultWindowSize, "the number of data points in the viewing window")
	refreshRate *int    = flag.Int("refreshRate", DefaultRefreshRate, "the number of data points to buffer before refreshing")
	maxFrames   *int    = flag.Int("maxFrames", DefaultMaxFrames, "maximum frames to read before turning off")
	mockDevice  *bool   = flag.Bool("mockDevice", false, "whether to use the mock device")
	dumpFrames  *bool   = flag.Bool("dumpFrames", false, "dump frames in Go format to frames.go")
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
	if err := device.Connect(); err != nil {
		log.Printf("Error: %v\n", err)
		return
	}
	out := device.Out()

	run(p, out)

	log.Printf("Thank you!")
	device.Disconnect()
}

func run(p *gplot.Plotter, out <-chan *DataFrame) {
	var (
		window1 = window.New(*windowSize, WindowMultiple)
		window2 = window.New(*windowSize, WindowMultiple)
	)

	var frames []*DataFrame
	// if dumpFrames is true, we will buffer the data
	// frames in memory and then dump them to frames.go
	// at the end
	if *dumpFrames {
		frames = make([]*DataFrame, 0)
		defer func() {
			var file *os.File
			var err error
			if *dumpFrames {
				file, err = os.OpenFile(DumpFile,
					os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
				if err != nil {
					log.Printf("could not prepare dump file: %v", err)
					return
				}
			}
			defer file.Close()
			// now output
			t := template.Must(template.New("").ParseFiles("etc/frames.template"))
			if err = t.ExecuteTemplate(file, "frames.template", frames); err != nil {
				log.Printf("error dumping to template: %v", err)
			}
		}()

	}

	for i := 0; i < *maxFrames; i++ {
		df, ok := <-out
		if !ok {
			log.Printf("The data channel got closed (exiting)")
			return
		}

		if *dumpFrames {
			if len(frames) < 1000 {
				frames = append(frames, df)
			}
		}

		log.Printf("Got df: %v", df.String())
		for _, v := range df.ChannelData(0) {
			window1.PushBack(v)
		}
		for _, v := range df.ChannelData(1) {
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
