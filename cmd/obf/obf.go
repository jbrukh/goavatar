//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package main

import (
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar/formats"
	"github.com/jbrukh/gplot"
	"os"
	//"time"
)

var (
	csv  *bool = flag.Bool("csv", false, "output strict CSV")
	plot *bool = flag.Bool("plot", false, "ouput the series on a gplot graph")
	seq  *bool = flag.Bool("seq", false, "read sequential data, if available")
)

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
# Endianness:     %d
# Reserved:       %x
# ------------------------------------------
`

//
// WARNING: this is a work in progress and only supports two channels for graphing.
//
func main() {
	flag.Parse()
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

	codec, err := NewObfReader(file)
	if err != nil {
		fmt.Printf("could not read the header")
		return
	}
	header := codec.Header()

	if !*csv {
		// format the header
		fmt.Printf(headerFmt, header.DataType, header.FormatVersion,
			header.StorageMode, header.Channels, header.Samples, header.SampleRate, header.Endianness, header.Reserved)
	}

	if *seq {
		v, ts, err := codec.Sequential()
		if err != nil {
			fmt.Printf("could not read sequential data: %v", err)
			return
		}
		for _, channel := range v {
			fmt.Printf("%v\n", channel)
		}
		fmt.Printf("%v\n", ts)
	} else {
		printParallel(codec)
	}
	if *plot {
		// read the data as a data frame
		channels, ts, err := codec.Sequential()
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
		p.CheckedCmd(fmt.Sprintf("set xrange [0:%v]", len(ts)))
		p.CheckedCmd("set terminal aqua size 1430,400")

		ch := len(channels)
		if ch == 1 {
			p.PlotX(channels[0], "Ch1")
		} else if ch == 2 {
			p.Dual(channels[0], channels[1], "Ch1", "Ch2")
		} else {
			fmt.Printf("sorry, max 2 channels is currently supported")
			return
		}
	}
}

func printParallel(codec ObfReader) {
	var (
		header  = codec.Header()
		ch      = int(header.Channels)
		samples = int(header.Samples)
	)
	fmt.Print("timestamp")
	for i := 0; i < int(header.Channels); i++ {
		fmt.Printf(",channel%d", i+1)
	}
	fmt.Println()

	bb, err := codec.Parallel()
	if err != nil {
		fmt.Println("ERR: could not get parallel format")
		os.Exit(1)
	}

	for j := 0; j < samples; j++ {
		// read each block
		values, ts := bb.Sample(0)
		fmt.Printf("%v", ts)
		for i := 0; i < ch; i++ {
			fmt.Printf(",%.20f", values[i])
		}
		fmt.Println()
	}
}
