//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package formats

import (
	"bytes"
	"encoding/binary"
	"fmt"
	. "github.com/jbrukh/goavatar"
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
//    + int64*samples):                  parallel format; blocks of channel
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
	OBFHeaderSize    = 31
	OBFTimestampSize = 4
	OBFValueSize     = 8
)

// Fixed locations
const (
	OBFHeaderAddr = 0
	OBFValuesAddr = OBFHeaderSize
)

var ByteOrder = binary.BigEndian

// ----------------------------------------------------------------- //
// TYPES
// ----------------------------------------------------------------- //

type (
	// The OBF Header, which keeps track
	// of versioning information as well
	// as the size of the data.
	OBFHeader struct {
		DataType      byte
		FormatVersion byte
		StorageMode   byte
		Channels      uint8
		Samples       uint32
		SampleRate    uint16
		Endianness    byte
		Reserved      [20]byte // reserved for extentions
	}

	// obfCodec will read and write the OBF
	// format on various levels of abstraction.
	obfCodec struct {
		file        io.ReadWriteSeeker
		header      OBFHeader
		payloadSize int64
	}

	OBFReader interface {
		// TODO: deprecate
		ReadParallelBlock() ([]float64, uint32, error)

		Header() *OBFHeader
		Parallel() (*BlockBuffer, error)
		//Sequential() ([][]float64, []uint32, error)
	}

	OBFWriter interface {
		SeekHeader() error
		SeekValues() error
		SeekParallel() error
		SeekSequential() error
		SeekSample(n int) error
	}
)

// Create a new obfCodec and read the header. If the header
// cannot be read an error is returned.
func newObfCodec(file io.ReadWriteSeeker) (oc *obfCodec, err error) {
	oc = &obfCodec{file: file}
	if err = oc.ReadHeader(); err != nil {
		return
	}
	oc.pyldSize(int64(oc.header.Samples), int64(oc.header.Channels))
	return
}

func NewOBFReader(file io.ReadWriteSeeker) (r OBFReader, err error) {
	return newObfCodec(file)
}

// ----------------------------------------------------------------- //
// Private Methods
// ----------------------------------------------------------------- //

func (oc *obfCodec) pyldSize(channels, samples int64) {
	oc.payloadSize = samples * (channels*
		OBFValueSize + OBFTimestampSize)
}

// Read a piece of binary data from the underlying stream.
func (oc *obfCodec) read(i interface{}) error {
	return binary.Read(oc.file, ByteOrder, i)
}

// Write a piece of binary data to the underlying stream,
// in place.
func (oc *obfCodec) write(i interface{}) error {
	return binary.Write(oc.file, ByteOrder, i)
}

// Read a block in place.
func (oc *obfCodec) readBlock(v []float64, ts *uint32) (err error) {
	if err = oc.read(v); err != nil {
		return
	}
	return oc.read(ts)
}

// Write a block in place.
func (oc *obfCodec) writeBlock(v []float64, ts uint32) (err error) {
	if err = oc.write(v); err != nil {
		return
	}
	return oc.write(ts)
}

// ----------------------------------------------------------------- //
// Seeking Operations
// ----------------------------------------------------------------- //

// Go to the starting position of the header.
func (oc *obfCodec) SeekHeader() (err error) {
	_, err = oc.file.Seek(OBFHeaderAddr, os.SEEK_SET)
	return
}

// Go to the starting position of the values.
func (oc *obfCodec) SeekValues() (err error) {
	_, err = oc.file.Seek(OBFValuesAddr, os.SEEK_SET)
	return
}

// Go to the starting position of the parallel values.
func (oc *obfCodec) SeekParallel() (err error) {
	_, err = oc.file.Seek(OBFValuesAddr, os.SEEK_SET)
	return
}

// Go to the starting position of the sequential values.
// TODO this will fail silently without having called ReadHeader().
func (oc *obfCodec) SeekSequential() (err error) {
	_, err = oc.file.Seek(OBFHeaderSize+oc.payloadSize, os.SEEK_SET)
	return
}

// Seek the n-th sample.
func (oc *obfCodec) SeekSample(n int) (err error) {
	panic("implement")
	return
}

// ----------------------------------------------------------------- //
// Reading Operations -- all these operations happen in-place
// ----------------------------------------------------------------- //

// Read the OBFHeader of this file.
func (oc *obfCodec) ReadHeader() (err error) {
	return oc.read(&oc.header) // TODO check valid header
}

// TODO:  deprecate
func (oc *obfCodec) ReadParallelBlock() (values []float64, ts uint32, err error) {
	if oc.header.StorageMode != StorageModeParallel {
		return nil, 0, fmt.Errorf("can only seek samples in parallel mode")
	}
	ch := int(oc.header.Channels)
	values = make([]float64, ch)

	err = binary.Read(oc.file, ByteOrder, values)
	if err != nil {
		return
	}

	err = binary.Read(oc.file, ByteOrder, &ts)
	return
}

// ----------------------------------------------------------------- //
// Reading Operations -- these operations seek also
// ----------------------------------------------------------------- //

// Return the last header that had been read. Notice
// header is read upon instantiation.
func (oc *obfCodec) Header() *OBFHeader {
	return &oc.header
}

// Read the entire set of parallel values from the file.
func (oc *obfCodec) Parallel() (b *BlockBuffer, err error) {
	if err = oc.SeekValues(); err != nil {
		return
	}
	header := oc.Header()
	channels, samples := int(header.Channels), int(header.Samples)
	b = NewBlockBuffer(channels, samples)
	v := make([]float64, channels)
	var ts uint32
	for s := 0; s < samples; s++ {
		oc.readBlock(v, &ts)
		b.AppendSample(v, int64(ts))
	}
	return
}

// ----------------------------------------------------------------- //
// Writing Operations -- All these operations happen in-place
// ----------------------------------------------------------------- //

// Write a new header to this file.
func (oc *obfCodec) WriteHeader(h *OBFHeader) (err error) {
	// go to the start of the file
	if err = oc.SeekHeader(); err != nil {
		return err
	}
	return oc.write(h)
}

// Writes a data frame in parallel mode, assuming the writer
// is at the correct location for the frame.
func (oc *obfCodec) WriteParallel(b *BlockBuffer, firstTs int64) (err error) {
	var (
		samples = b.Samples()
	)

	buf := new(bytes.Buffer)
	for i := 0; i < samples; i++ {
		v, ts := b.NextSample()
		oc.writeBlock(v, uint32((ts-firstTs)/1000000))
	}

	//log.Printf("writing parallel blocks: %v", buf.Bytes())
	return oc.write(buf.Bytes())
	//log.Printf("finished: %v", err)
	return
}
