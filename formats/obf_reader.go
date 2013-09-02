//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package formats

import (
	"encoding/binary"
	. "github.com/jbrukh/goavatar/datastruct"
	"io"
)

type (
	// obfCodec will read and write the OBF
	// format on various levels of abstraction.
	obfReader struct {
		r  io.Reader
		h  ObfHeader
		ps int64 // payload size
	}
)

// NewObfReader creates a vanilla OBF deserializer that reads
// the OBF stream sequentially.
func NewObfReader(r io.Reader) (ObfReader, error) {
	or := &obfReader{r: r}

	// first things first, read the header
	if err := binary.Read(or.r, ByteOrder, &or.h); err != nil {
		return nil, err
	}

	// set the payload size
	or.ps = getPayloadSize(int64(or.h.Channels), int64(or.h.Samples))
	return or, nil
}

func (or *obfReader) Header() *ObfHeader {
	return nil // TODO
}

func (or *obfReader) Parallel() (*BlockBuffer, error) {
	return nil, nil // TODO
}

func (or *obfReader) Sequential() ([][]float64, []int64, error) {
	return nil, nil, nil // TODO
}
