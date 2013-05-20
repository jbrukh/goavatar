//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package etc

import (
	. "github.com/jbrukh/goavatar/device"
	. "github.com/jbrukh/goavatar/formats"
	"log"
	"os"
)

// Read an OBF file as an array of generic
// data frames. Mostly for use with the mock
// device.
func MockDataFrames(fn string) (d []DataFrame, err error) {
	file, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	codec, err := NewOBFReader(file)
	if err != nil {
		return nil, err
	}

	b, err := codec.Parallel()
	if err != nil {
		return
	}

	for b.Samples() > 0 {
		var (
			bb = b.PopDownSample(16)
			df = NewDataFrame(bb, 250)
		)
		d = append(d, df)
	}

	log.Printf("loaded mock data with %d data frames", len(d))
	return d, nil
}
