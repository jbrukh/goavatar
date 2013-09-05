//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package obf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	. "github.com/jbrukh/goavatar/datastruct"
	"io"
	"os"
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
	FormatVersion1   = 0x01 // in this format, we have a 10 byte header
	FormatVersion2   = 0x02 // in this format, we add a field for Endianness and 20 bytes of padding
	FormatVersion2_1 = 0x03 // in this format, we add an IndexUnit field
)

// Default format version
const ObfDefaultFormatVersion = FormatVersion2_1

// Endianness
const (
	BigEndian    = 0x00
	LittleEndian = 0x01
)

// Default byte order
const ObfDefaultByteOrder = BigEndian

// IndexUnit
const (
	UnitMilliseconds = 0x00
	UnitNanoseconds  = 0x01
	UnitSeconds      = 0x02
	UnitHertz        = 0x03
	UnitEnumeration  = 0x04 // just monotonically increasing integers
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
		Write(*BlockBuffer) error
		Close() error
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
// OBF Header
// ----------------------------------------------------------------- //

func (h *ObfHeader) Dim() (channels, samples int) {
	return int(h.Channels), int(h.Samples)
}

// ----------------------------------------------------------------- //
// Generic Reading Methods -- all these read the current position
// ----------------------------------------------------------------- //

// ReadHeader will read in the OBF header from the underlying
// reader. This function assumes that the pointer of the reader
// is pointing to the start of the header.
func ReadHeader(r io.Reader) (header *ObfHeader, err error) {
	header = new(ObfHeader)
	if err := binary.Read(r, ByteOrder, header); err != nil {
		return nil, err
	}
	return
}

// ReadParallel will read in the parallel data payload. This
// function assumes that the pointer of the reader is pointing
// to the start of the data. If the OBF file does not support
// parallel data, then an error is returned.
func ReadParallel(r io.Reader, header *ObfHeader) (*BlockBuffer, error) {
	if header.StorageMode == StorageModeSequential {
		return nil, fmt.Errorf("no parallel payload, use sequential")
	}
	var (
		channels, samples = header.Dim()
		b                 = NewBlockBuffer(channels, samples)
		v                 = make([]float64, channels)
		inx32             uint32
	)
	for s := 0; s < samples; s++ {
		if err := readBlock(r, v, &inx32); err != nil {
			return nil, err
		}
		b.AppendSample(v, ToTs64(inx32))
	}
	return b, nil
}

// ReadDSequential will read in the sequential data payload. This
// function assumes that the pointer of the reader is pointing
// to the start of the data. If the file does not support
// sequential data, then an error is returned.
func ReadSequential(r io.Reader, header *ObfHeader) (v [][]float64, inxs []int64, err error) {
	if header.StorageMode == StorageModeParallel {
		return nil, nil, fmt.Errorf("no sequential payload, use parallel")
	}

	channels, samples := header.Dim()
	v = make([][]float64, channels)

	// read in all the channels sequentially
	for c := 0; c < channels; c++ {
		v[c] = make([]float64, samples)
		if err = binary.Read(r, ByteOrder, v[c]); err != nil {
			return nil, nil, err
		}
	}

	// allocate the indices
	inxs = make([]int64, samples)

	// read and convert all the indices
	for s := 0; s < samples; s++ {
		var inx32 uint32
		if err = binary.Read(r, ByteOrder, &inx32); err != nil {
			return nil, nil, err
		}
		inxs[s] = ToTs64(inx32)
	}
	return
}

// ----------------------------------------------------------------- //
// Generic Writing Methods -- all these write the current position
// ----------------------------------------------------------------- //

func WriteHeader(w io.Writer, header *ObfHeader) (err error) {
	return binary.Write(w, ByteOrder, header)
}

func WriteParallel(w io.Writer, b *BlockBuffer, indexFunc func(int64) uint32) (err error) {
	// write parallel samples to a buffer
	buf := new(bytes.Buffer)
	samples := b.Samples()

	for s := 0; s < samples; s++ {
		v, ts := b.Sample(s)
		if err = writeBlock(buf, v, indexFunc(ts)); err != nil {
			return
		}
	}

	//log.Printf("writing parallel blocks: %v", buf.Bytes())
	return binary.Write(w, ByteOrder, buf.Bytes())
}

func WriteSequential(w io.Writer, b *BlockBuffer, indexFunc func(int64) uint32) (err error) {
	arr, ts64 := b.Arrays()
	for _, channel := range arr {
		if err = binary.Write(w, ByteOrder, channel); err != nil {
			return
		}
	}
	ts32 := make([]uint32, len(ts64))
	for i, tv := range ts64 {
		ts32[i] = indexFunc(tv)
	}
	return binary.Write(w, ByteOrder, ts32)
}

// ----------------------------------------------------------------- //
// Duration Methods
// ----------------------------------------------------------------- //

func ApproxDurationMs(obfFile string) (d int, err error) {
	r, err := os.Open(obfFile)
	if err != nil {
		return
	}
	h, err := ReadHeader(r)
	if err != nil {
		return
	}
	d = 1000 * int(h.Samples) / int(h.SampleRate)
	return
}

// ----------------------------------------------------------------- //
// Private Helper Methods
// ----------------------------------------------------------------- //

// getPayloadSize calculates the size of the payload based on the
// number of channels and index values.
func getPayloadSize(channels, samples int) int64 {
	return int64(samples) * (int64(channels)*ObfValueSize + ObfIndexValueSize)
}

func ToTs64(ts uint32) int64 {
	return int64(ts) * 1000000
}

func ToTs32(ts int64) uint32 {
	return uint32(ts / 1000000)
}

func ToTs32Diff(ts int64, diff int64) uint32 {
	return ToTs32(ts - diff)
}

func writeBlock(w io.Writer, v []float64, ts uint32) (err error) {
	if err = binary.Write(w, ByteOrder, v); err != nil {
		return
	}
	return binary.Write(w, ByteOrder, ts)
}

// Read a block in place.
func readBlock(r io.Reader, v []float64, ts *uint32) (err error) {
	if err = binary.Read(r, ByteOrder, &v); err != nil {
		return
	}
	return binary.Read(r, ByteOrder, ts)
}
