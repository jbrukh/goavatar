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
	or.ps = getPayloadSize(or.header.Ch(), or.header.S())
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

	// create the
	var (
		//	v     = make([]float64, or.h.Ch())
		b = NewBlockBuffer(or.header.Ch(), or.header.S())
	//	inx32 uint32
	)

	// err = oc.forSamples(func(s int) (err error) {
	// 	if err = oc.readBlock(v, &ts32); err != nil {
	// 		return
	// 	}
	// 	b.AppendSample(v, toTs64(ts32))
	// 	return
	// })
	return b, nil
}

func (or *obfReader) Sequential() ([][]float64, []int64, error) {
	if or.read {
		return nil, nil, fmt.Errorf("stream exhausted (get a new reader)")
	} else {
		panic("implement me")
	}
}
