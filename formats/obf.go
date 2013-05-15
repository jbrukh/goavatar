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
	//"log"
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
		ReadParallelBlock() ([]float64, int64, error)

		Header() *OBFHeader
		Parallel() (*BlockBuffer, error)
		Sequential() ([][]float64, []int64, error)
	}

	OBFWriter interface {
		SeekHeader() error
		SeekValues() error
		SeekParallel() error
		SeekSequential() error
		SeekSample(n int) error

		WriteHeader(*OBFHeader) error
		WriteParallel(*BlockBuffer, int64) error
	}
)

// Create a new obfCodec.
func newObfCodec(file io.ReadWriteSeeker) (oc *obfCodec) {
	return &obfCodec{file: file}
}

// Create a new OBFReader and read the header. If the header
// cannot be read an error is returned.
func NewOBFReader(file io.ReadWriteSeeker) (r OBFReader, err error) {
	oc := newObfCodec(file)
	if err = oc.ReadHeader(); err != nil {
		return
	}
	oc.pyldSize(int64(oc.samples()), int64(oc.channels()))
	return oc, nil
}

// ----------------------------------------------------------------- //
// Helper Methods
// ----------------------------------------------------------------- //

func toTs(ts uint32) int64 {
	return int64(ts) * 1000000
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

// ----------------------------------------------------------------- //
// Private Methods
// ----------------------------------------------------------------- //

func (oc *obfCodec) pyldSize(channels, samples int64) {
	oc.payloadSize = samples * (channels*
		OBFValueSize + OBFTimestampSize)
}

// Validate the last header that has been read.
func (oc *obfCodec) validate() (err error) {
	return // TODO TODO TODO
}

// Read a piece of binary data from the underlying stream.
func (oc *obfCodec) read(i interface{}) error {
	return binary.Read(oc.file, ByteOrder, i)
}

// Write a piece of binary data to the underlying stream,
// in place.
func (oc *obfCodec) write(i interface{}) error {
	return writeTo(oc.file, i)
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
	return writeBlockTo(oc.file, v, ts)
}

// Return the last read number of channels.
func (oc *obfCodec) channels() int {
	return int(oc.header.Channels)
}

// Return the last read number of samples.
func (oc *obfCodec) samples() int {
	return int(oc.header.Samples)
}

// Return the storage mode as an integer.
func (oc *obfCodec) mode() byte {
	return oc.header.StorageMode
}

// Return a new buffer big enough for this
// OBF file.
func (oc *obfCodec) buffer() *BlockBuffer {
	return NewBlockBuffer(oc.channels(), oc.samples())
}

// Return a slice big enough to hold one block
// of channel values.
func (oc *obfCodec) block() []float64 {
	return make([]float64, oc.channels())
}

func (oc *obfCodec) channel() []float64 {
	return make([]float64, oc.samples())
}

func (oc *obfCodec) timestamps() []int64 {
	return make([]int64, oc.samples())
}

func (oc *obfCodec) forChannels(f func(c int) error) error {
	channels := oc.channels()
	for c := 0; c < channels; c++ {
		if err := f(c); err != nil {
			return err
		}
	}
	return nil
}

func (oc *obfCodec) forSamples(f func(s int) error) error {
	samples := oc.samples()
	for s := 0; s < samples; s++ {
		if err := f(s); err != nil {
			return err
		}
	}
	return nil
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
	if oc.mode() == StorageModeSequential {
		return fmt.Errorf("no parallel values available in this mode")
	}
	_, err = oc.file.Seek(OBFValuesAddr, os.SEEK_SET)
	return
}

// Go to the starting position of the sequential values.
// TODO this will fail silently without having called ReadHeader().
func (oc *obfCodec) SeekSequential() (err error) {
	if oc.mode() == StorageModeParallel {
		return fmt.Errorf("no sequential values available in this mode")
	}
	_, err = oc.file.Seek(OBFHeaderSize+oc.payloadSize, os.SEEK_SET)
	return
}

// Seek the n-th sample.
func (oc *obfCodec) SeekSample(n int) (err error) {
	if oc.mode() == StorageModeSequential {
		return fmt.Errorf("no parallel values available in this mode")
	}
	panic("implement")
	return
}

// ----------------------------------------------------------------- //
// Reading Operations -- all these operations happen in-place
// ----------------------------------------------------------------- //

// Read the OBFHeader of this file.
func (oc *obfCodec) ReadHeader() (err error) {
	if err = oc.read(&oc.header); err != nil {
		return
	}
	return oc.validate()
}

// Read the entire set of parallel values from
// the file starting at the current position.
func (oc *obfCodec) ReadParallel() (b *BlockBuffer, err error) {
	var (
		v    = oc.block()
		ts32 uint32
	)
	b = oc.buffer()
	err = oc.forSamples(func(s int) (err error) {
		if err = oc.readBlock(v, &ts32); err != nil {
			return
		}
		b.AppendSample(v, toTs(ts32))
		return
	})
	return
}

func (oc *obfCodec) ReadSequential() (v [][]float64, ts []int64, err error) {
	// allocate channel slices
	v = make([][]float64, oc.channels())

	// read in all the channels sequentially
	err = oc.forChannels(func(c int) (err error) {
		v[c] = oc.channel()
		return oc.read(v[c])
	})
	if err != nil {
		return
	}

	// allocate the timestamps
	ts = oc.timestamps()

	// read and convert all the timestamps
	err = oc.forSamples(func(s int) (err error) {
		var ts32 uint32
		if err = oc.read(&ts32); err != nil {
			return
		}
		ts[s] = toTs(ts32)
		return
	})
	return
}

// TODO:  deprecate
func (oc *obfCodec) ReadParallelBlock() (values []float64, ts int64, err error) {
	values = oc.block()
	if err = oc.read(values); err != nil {
		return
	}
	var ts32 uint32
	err = oc.read(&ts32)
	ts = toTs(ts32)
	return
}

// ----------------------------------------------------------------- //
// Reading Operations -- these operations seek and do validation,
// so are more user-facing and safer
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
	return oc.ReadParallel()
}

// Read the entire set of sequential values from the file.
func (oc *obfCodec) Sequential() (v [][]float64, ts []int64, err error) {
	if err = oc.SeekSequential(); err != nil {
		return
	}
	return oc.ReadSequential()
}

// ----------------------------------------------------------------- //
// Writing Operations -- All these operations happen in-place
// ----------------------------------------------------------------- //

// Write a new header to this file.
func (oc *obfCodec) WriteHeader(h *OBFHeader) (err error) {
	return oc.write(h)
}

// Writes a data frame in parallel mode, assuming the writer
// is at the correct location for the frame.
func (oc *obfCodec) WriteParallel(b *BlockBuffer, tsTransform func(int64) uint32) (err error) {
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
	return oc.write(buf.Bytes())
}
