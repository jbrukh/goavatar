//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package formats

import (
	"encoding/binary"
	"fmt"
	. "github.com/jbrukh/goavatar/datastruct"
	"io"
)

type (
	// obfCodec will read and write the OBF
	// format on various levels of abstraction.
	obfReader struct {
		r      io.Reader
		header ObfHeader
		ps     int64        // payload size
		b      *BlockBuffer // data read from parallel payload
		read   bool         // whether the stream is exhausted
	}
)

// NewObfReader creates a vanilla OBF deserializer that reads
// the OBF stream sequentially.
func NewObfReader(r io.Reader) (ObfReader, error) {
	or := &obfReader{r: r}
	if err := binary.Read(or.r, ByteOrder, &or.header); err != nil {
		return nil, err
	}
	or.ps = getPayloadSize(or.header.Dim())
	return or, nil
}

// Header returns the header of this OBF stream, which
// is always available upon construction. This function
// does not have any effect on internal state.
func (or *obfReader) Header() *ObfHeader {
	return &or.header
}

// Parallel returns the data by reading the parallel
// payload. If the stream was exhausted
// by reading the sequential format first, then this
// method will return an error. This method may only
// be called once.
func (or *obfReader) Parallel() (*BlockBuffer, error) {
	// only read the stream once
	if or.read {
		return nil, fmt.Errorf("stream exhausted (get a new reader)")
	}
	or.read = true
	return ReadParallel(or.r, &or.header)
}

func (or *obfReader) Sequential() (v [][]float64, inxs []int64, err error) {
	if or.read {
		return nil, nil, fmt.Errorf("stream exhausted (get a new reader)")
	}
	or.read = true

	// if the storage mode is combined, we must skip to the
	// start of the sequential payload
	if or.header.StorageMode == StorageModeCombined {
		// throw away the parallel
		if _, err = io.ReadFull(or.r, make([]byte, or.ps)); err != nil {
			return nil, nil, err
		}
	}
	return ReadSequential(or.r, &or.header)
}
