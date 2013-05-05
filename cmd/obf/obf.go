//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package main

import (
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar/formats"
	"os"
	"time"
)

var humanTime *bool = flag.Bool("humanTime", false, "format timestamps")

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
# ------------------------------------------
`

func main() {
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("usage: obf [opts] [file]")
		return
	}
	fileName := args[0]

	file, err := os.OpenFile(fileName, os.O_RDONLY, 0655)
	if err != nil {
		fmt.Printf("could not open file: %v\n", err)
		return
	}
	defer file.Close()

	fmt.Println(preludeFmt)

	codec := NewOBFCodec(file)
	header, err := codec.ReadHeader()
	if err != nil {
		fmt.Printf("could not read the header: %v\n", err)
		return
	}

	// format the header
	fmt.Printf(headerFmt, header.DataType, header.FormatVersion, header.StorageMode, header.Channels, header.Samples)

	// format the data
	fmt.Print("timestamp")
	for i := 0; i < int(header.Channels); i++ {
		fmt.Printf(", channel%d", i+1)
	}
	fmt.Println()

	ch, samples := int(header.Channels), int(header.Samples)
	for j := 0; j < samples; j++ {
		// read each block
		values, ts, err := codec.ReadParallelBlock()
		if err != nil {
			fmt.Printf("could not read block: %v\n", err)
			return
		}

		// print the values and timestamp
		if *humanTime {
			nsec := ts % 1000000000
			sec := (ts - nsec) / 1000000000
			human := time.Unix(sec, nsec).Format(time.RFC3339Nano)
			fmt.Printf("%v", human)
		} else {
			fmt.Printf("%v", ts)
		}
		for i := 0; i < ch; i++ {
			fmt.Printf(", %.25f", values[i])
		}
		fmt.Println()
	}
}
