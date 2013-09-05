//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package obf

import (
	"fmt"
	. "github.com/jbrukh/goavatar/datastruct"
	"io"
)

type (
	// obfCodec will read and write the OBF
	// format on various levels of abstraction.
	obfWriter struct {
		out          io.Writer
		buf          *BlockBuffer
		indexFunc    func(int64) uint32
		storageMode  byte
		channels     int
		sampleRate   int
		indexUnit    byte
		doParallel   bool
		doSequential bool
	}
)

// NewObfWriter creates a new OBF writer. The writer will first
// collect a bunch of BlockBuffers full of data. Upon calling Close(),
// the writer will write them to the provided output writer with the
// appropriate storage mode.
func NewObfWriter(w io.Writer, indexFunc func(int64) uint32, storageMode byte, sampleRate int, indexUnit byte) (or ObfWriter) {
	return &obfWriter{
		out:          w,
		indexFunc:    indexFunc,
		storageMode:  storageMode,
		channels:     -1,
		sampleRate:   sampleRate,
		indexUnit:    indexUnit,
		doParallel:   storageMode == StorageModeParallel || storageMode == StorageModeCombined,
		doSequential: storageMode == StorageModeSequential || storageMode == StorageModeCombined,
	}
}

// Write will write a BlockBuffer to this writer. This method may be
// called multiple times.
func (ow *obfWriter) Write(b *BlockBuffer) (err error) {
	if ch := b.Channels(); ow.channels < 0 {
		ow.buf = NewBlockBuffer(ch, b.Samples())
	} else if ow.channels != ch {
		return fmt.Errorf("expecting a buffer with %d channels", ow.channels)
	}
	ow.buf.Append(b)
	return
}

func (ow *obfWriter) Close() (err error) {

	header := &ObfHeader{
		DataType:      DataTypeRaw,
		FormatVersion: ObfDefaultFormatVersion,
		StorageMode:   ow.storageMode,
		Channels:      uint8(ow.buf.Channels()),
		Samples:       uint32(ow.buf.Samples()),
		SampleRate:    uint16(ow.sampleRate),
		Endianness:    ObfDefaultByteOrder,
	}

	if err = WriteHeader(ow.out, header); err != nil {
		return
	}

	if ow.doParallel {
		if err = WriteParallel(ow.out, ow.buf, ow.indexFunc); err != nil {
			return
		}
	}

	if ow.doSequential {
		if err = WriteSequential(ow.out, ow.buf, ow.indexFunc); err != nil {
			return
		}
	}
	return
}
