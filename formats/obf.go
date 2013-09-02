//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package formats

import (
	"bytes"
	"encoding/binary"
	"io"
	//"log"
	. "github.com/jbrukh/goavatar/datastruct"
)

// ----------------------------------------------------------------- //
// Octopus Binary Format (OBF) Version 1
// (Parallel Only)
//
// Header (10 bytes):
//    DataType (1 byte):                 0x01 = raw device data;
//    FormatVersion (1 byte):            0x01 = version 1
//    StorageMode (1 byte):              0x01 = parallel; 0x02 = sequential
//    Channels (1 byte):                 0-255 channels
//    Samples (uint32):                  number of samples stored
//    SampleRate (uint16):               the sample rate at which this
//                                       data was sampled
//
// Payload (variable):
//    Values + Timestamps
//    (float64*channels*samples
//    + uint32*samples):                  parallel format; blocks of channel
//                                       values + timestamps
//
// ----------------------------------------------------------------- //
// Octopus Binary Format (OBF) Version 2
// (Combined, 32-bit relative timestamps)
//
// Header (31 bytes):
//    DataType (1 byte):                 0x01 = raw device data;
//    FormatVersion (1 byte):            0x01 = version 1
//    StorageMode (1 byte):              0x01 = parallel; 0x02 = sequential;
//                                       0x03 = combined
//    Channels (1 byte):                 0-255 channels
//    Samples (uint32):                  number of samples stored
//    SampleRate (uint16):               the sample rate at which this
//                                       data was sampled
//    Endianness (1 byte):               0x00 = Big; 0x01 = Little
//    Reserved (20 bytes):               reserved for future expansions
//
// P-mode Values (variable):
//    Values + Timestamps
//    (float64*channels*samples
//    + uint32*samples):                  parallel format; blocks of channel
//                                        values + timestamps (in ms starting at 0)
//
// S-mode Values (variable):
//    Values (float64*channels*samples):  sequential format
//    Timestamps (uint32*samples):        timestamps of the values (unsigned,
//                                        in ms starting at 0)
//
// ----------------------------------------------------------------- //
// Octopus Binary Format (OBF) Version 2.1
// (Adding independent variable unit)
//
// Header will now be 32 bytes, with 19 bytes reserved. The 12th byte
// of the header will be the unit of the index variable. This version
// is backwards compatible with 2.0, taking the unit of the index
// variable to be milliseconds by default.
//
// ----------------------------------------------------------------- //
// Notes on P-mode vs S-mode:
//
// Define v(c,s) to mean the value of channel c (0 < c <= C) at
// sample s (0 <= s < S) where C is the number of channels and
// S is the number of samples. Define T(s) to mean the timestamp
// at time of sample s.
//
// Then "parallel" mode is:
//
//    concat[v(1,s), ..., v(C,s), T(s)] for all t.
//
// For "sequential" mode:
//
//    concat[v(c,0), ..., v(c,S-1)] for all c, followed by
//    [T(s)] for all s.
//

// ----------------------------------------------------------------- //
// FIELD VALUES
// ----------------------------------------------------------------- //

// DataTypes
const (
	DataTypeRaw = 0x01
)

// FormatVersions
const (
	FormatVersion1 = 0x01 // in this format, we have a 10 byte header
	FormatVersion2 = 0x02 // in this format, we add a field for Endianness and 20 bytes of padding
)

// Endianness
const (
	BigEndian    = 0x00
	LittleEndian = 0x01
)

// IndexUnit
const (
	Milliseconds = 0x00
	Nanoseconds  = 0x01
	Seconds      = 0x02
	Hertz        = 0x03
	Enumeration  = 0x04 // just monotonically increasing integers
)

// StorageModes
const (
	StorageModeParallel   = 0x01
	StorageModeSequential = 0x02
	StorageModeCombined   = 0x03
)

// ----------------------------------------------------------------- //
// SIZES
// ----------------------------------------------------------------- //

//
// IF YOU ARE MODIFYING THE FORMAT, MAKE SURE
// TO ADJUST THESE. Sizes of the header and
// data point sizes.
//
const (
	ObfHeaderSize     = 31
	ObfIndexValueSize = 4
	ObfValueSize      = 8
)

// Fixed locations
const (
	ObfHeaderAddr = 0
	OBFValuesAddr = ObfHeaderSize
)

var ByteOrder = binary.BigEndian

// ----------------------------------------------------------------- //
// TYPES
// ----------------------------------------------------------------- //

type (
	// The OBF Header, which keeps track
	// of versioning information as well
	// as the size of the data.
	ObfHeader struct {
		DataType      byte
		FormatVersion byte
		StorageMode   byte
		Channels      uint8
		Samples       uint32
		SampleRate    uint16
		Endianness    byte
		IndexUnit     byte
		Reserved      [19]byte // reserved for extentions
	}

	// ObfReader can read OBF files. Depending on
	// the implementation it may or may not be able
	// to seek to parts of the file.
	ObfReader interface {
		Header() *ObfHeader
		Parallel() (*BlockBuffer, error)
		Sequential() ([][]float64, []int64, error)
	}

	// ObfWriter can write OBF files. Depending on
	// the implementation it may or may not be able
	// to seek to parts of the file.
	ObfWriter interface {
		WriteHeader(*ObfHeader) error
		WriteParallel(*BlockBuffer, func(int64) uint32) error
	}

	// ObfSeeker is able to seek to sections of OBF.
	// If implementor is also an ObfReader or ObfWriter
	// it may also be able to read or write those
	// sections.
	ObfSeeker interface {
		SeekHeader() error
		SeekValues() error
		SeekParallel() error
		SeekSequential() error
		SeekSample(n int) error
	}
)

// ----------------------------------------------------------------- //
// Helper Methods
// ----------------------------------------------------------------- //

// getPayloadSize calculates the size of the payload based on the
// number of channels and index values.
func getPayloadSize(channels, samples int64) int64 {
	return samples * (channels*ObfValueSize + ObfIndexValueSize)
}

func toTs64(ts uint32) int64 {
	return int64(ts) * 1000000
}

func toTs32(ts int64) uint32 {
	return uint32(ts / 1000000)
}

func toTs32Diff(ts int64, diff int64) uint32 {
	return toTs32(ts - diff)
}

func writeTo(w io.Writer, i interface{}) error {
	return binary.Write(w, ByteOrder, i)
}

func writeBlockTo(w io.Writer, v []float64, ts uint32) (err error) {
	if err = writeTo(w, v); err != nil {
		return
	}
	return writeTo(w, ts)
}

func WriteParallelTo(w io.Writer, b *BlockBuffer, tsTransform func(int64) uint32) (err error) {
	// write parallel samples to a buffer
	buf := new(bytes.Buffer)
	samples := b.Samples()

	for s := 0; s < samples; s++ {
		v, ts := b.Sample(s)
		if err = writeBlockTo(buf, v, tsTransform(ts)); err != nil {
			return
		}
	}

	//log.Printf("writing parallel blocks: %v", buf.Bytes())
	return writeTo(w, buf.Bytes())
}

func (oc *obfCodec) WriteSequential(b *BlockBuffer, tsTransform func(int64) uint32) (err error) {
	return WriteSequentialTo(oc.file, b, tsTransform)
}

func WriteSequentialTo(w io.Writer, b *BlockBuffer, tsTransform func(int64) uint32) (err error) {
	arr, ts64 := b.Arrays()
	for _, channel := range arr {
		if err = writeTo(w, channel); err != nil {
			return
		}
	}
	ts32 := make([]uint32, len(ts64))
	for i, tv := range ts64 {
		ts32[i] = tsTransform(tv)
	}
	return writeTo(w, ts32)
}
