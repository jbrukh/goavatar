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
		header *ObfHeader
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
	return or.header
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
	} else {
		or.read = true
	}

	var (
		channels, samples = or.header.Dim()
		b                 = NewBlockBuffer(channels, samples)
		v                 = make([]float64, channels)
		inx32             uint32
	)

	for s := 0; s < samples; s++ {
		if err := readBlock(or.r, v, &inx32); err != nil {
			return nil, err
		}
		b.AppendSample(v, toTs64(inx32))
	}

	return b, nil
}

func (or *obfReader) Sequential() ([][]float64, []int64, error) {
	if or.read {
		return nil, nil, fmt.Errorf("stream exhausted (get a new reader)")
	} else {
		panic("implement me")
	}
}
