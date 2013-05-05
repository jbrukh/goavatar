//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package main

import (
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar"
	. "github.com/jbrukh/goavatar/formats"
	"github.com/jbrukh/gplot"
	"os"
	"time"
)

var (
	humanTime *bool = flag.Bool("humanTime", false, "format timestamps")
	csv       *bool = flag.Bool("csv", false, "output strict CSV")
	plot      *bool = flag.Bool("plot", false, "ouput the series on a gplot graph")
)

func init() {
	flag.Parse()
}

const preludeFmt = `# Octopus Binary Format.
#
# Copyright (c) 2013. Jake Brukhman/Octopus.
# All rights reserved.
#`

const headerFmt = `# HEADER ----------------------------------
# DataType:       %d
# FormatVersion:  %d
# StorageMode:    %d
# Channels:       %d
# Samples:        %d
# SampleRate:     %d
# ------------------------------------------
`

//
// WARNING: this is a work in progress and only supports two channels.
//
func main() {
	// read the options and args
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("usage: obf [opts] [file]")
		return
	}
	fileName := args[0]

	// open the file to read
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0655)
	if err != nil {
		fmt.Printf("could not open file: %v\n", err)
		return
	}
	defer file.Close()

	if !*csv {
		fmt.Println(preludeFmt)
	}
	codec := NewOBFCodec(file)
	header, err := codec.ReadHeader()
	if err != nil {
		fmt.Printf("could not read the header: %v\n", err)
		return
	}

	if !*csv {
		// format the header
		fmt.Printf(headerFmt, header.DataType, header.FormatVersion,
			header.StorageMode, header.Channels, header.Samples, header.SampleRate)
	}

	// format the data
	fmt.Print("timestamp")
	for i := 0; i < int(header.Channels); i++ {
		fmt.Printf(",channel%d", i+1)
	}
	fmt.Println()

	printFrames(codec)

	if *plot {
		// read the data as a data frame
		df, err := codec.ReadDataFrame()
		if err != nil {
			fmt.Printf("could not read the data as a frame: %v\n", err)
			return
		}

		p, err := gplot.NewPlotter(false)
		if err != nil {
			fmt.Printf("create the plot: %v\n", err)
			return
		}
		defer p.Close()

		//p.CheckedCmd("set yrange [0.01:0.018]")
		p.CheckedCmd(fmt.Sprintf("set xrange [0:%v]", df.Samples()))
		p.CheckedCmd("set terminal aqua size 1430,400")

		ch := df.Channels()
		buf := df.Buffer()
		if ch == 1 {
			p.PlotX(buf.ChannelData(0), "Ch1")
		} else if ch == 2 {
			p.Dual(buf.ChannelData(0), buf.ChannelData(1), "Ch1", "Ch2")
		} else {
			fmt.Printf("sorry, max 2 channels is currently supported")
			return
		}
	}
}

func printFrames(codec *OBFCodec) {
	var (
		header  = codec.Header()
		ch      = int(header.Channels)
		samples = int(header.Samples)
	)
	for j := 0; j < samples; j++ {
		// read each block
		values, ts, err := codec.ReadParallelBlock()
		if err != nil {
			fmt.Printf("could not read block: %v\n", err)
			return
		}

		// print the values and timestamp
		if *humanTime {
			human := NanosToTime(ts).Format(time.RFC3339Nano)
			fmt.Printf("%v", human)
		} else {
			fmt.Printf("%v", ts)
		}
		for i := 0; i < ch; i++ {
			fmt.Printf(",%.20f", values[i])
		}
		fmt.Println()
	}
}
