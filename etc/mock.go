//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package etc

import (
	. "github.com/jbrukh/goavatar"
	. "github.com/jbrukh/goavatar/formats"
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

	codec := NewOBFReader(file)
	b, err := codec.Parallel()
	if err != nil {
		return
	}

	for b.Samples() > 0 {
		var (
			bb = b.DownSample(16)
			df = NewDataFrame(bb, 250)
		)
		d = append(d, df)
	}

	return d, nil
}
