//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package formats

import (
	"encoding/binary"
	"fmt"
	. "github.com/jbrukh/goavatar/datastruct"
	"io"
	"os"
)

type (
	// ObfCodec is able to perform all reading, writing,
	// and seeking operations on an OBF file. This is
	// typically performed on hard files on the file system
	// since Go doesn't have an out-of-the-box in-memory
	// io.ReadWriteSeeker.
	ObfCodec interface {
		ObfReader
		ObfWriter
		ObfSeeker
	}

	// obfCodec will read and write the OBF
	// format on various levels of abstraction.
	obfCodec struct {
		file        io.ReadWriteSeeker
		header      ObfHeader
		payloadSize int64
	}
)

// Create a new OBFReader and read the header. If the header
// cannot be read an error is returned.
func NewObfCodec(file io.ReadWriteSeeker) (oc ObfCodec, err error) {
	c := &obfCodec{file: file}
	if err = c.ReadHeader(); err != nil {
		return
	}
	return c, nil
}

// ----------------------------------------------------------------- //
// Private Methods
// ----------------------------------------------------------------- //

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
	return binary.Write(oc.file, ByteOrder, i)
}

func (oc *obfCodec) timestamps() []int64 {
	_, samples := oc.header.Dim()
	return make([]int64, samples)
}

func (oc *obfCodec) forChannels(f func(c int) error) error {
	channels, _ := oc.header.Dim()
	for c := 0; c < channels; c++ {
		if err := f(c); err != nil {
			return err
		}
	}
	return nil
}

func (oc *obfCodec) forSamples(f func(s int) error) error {
	_, samples := oc.header.Dim()
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
	_, err = oc.file.Seek(ObfHeaderAddr, os.SEEK_SET)
	return
}

// Go to the starting position of the values.
func (oc *obfCodec) SeekValues() (err error) {
	_, err = oc.file.Seek(OBFValuesAddr, os.SEEK_SET)
	return
}

// Go to the starting position of the parallel values.
func (oc *obfCodec) SeekParallel() (err error) {
	if oc.header.StorageMode == StorageModeSequential {
		return fmt.Errorf("no parallel values available in this mode")
	}
	_, err = oc.file.Seek(OBFValuesAddr, os.SEEK_SET)
	return
}

// Go to the starting position of the sequential values.
// TODO this will fail silently without having called ReadHeader().
func (oc *obfCodec) SeekSequential() (err error) {
	if oc.header.StorageMode == StorageModeParallel {
		return fmt.Errorf("no sequential values available in this mode")
	}
	_, err = oc.file.Seek(ObfHeaderSize+oc.payloadSize, os.SEEK_SET)
	return
}

// Seek the n-th sample.
func (oc *obfCodec) SeekSample(n int) (err error) {
	if oc.header.StorageMode == StorageModeSequential {
		return fmt.Errorf("no parallel values available in this mode")
	}
	panic("implement")
}

// ----------------------------------------------------------------- //
// Reading Operations -- all these operations happen in-place
// ----------------------------------------------------------------- //

// Read the ObfHeader of this file.
func (oc *obfCodec) ReadHeader() (err error) {
	if err = oc.read(&oc.header); err != nil {
		return
	}
	//	channels, samples := oc.header.Dim()
	oc.payloadSize = getPayloadSize(oc.header.Dim())
	return oc.validate()
}

// Read the entire set of parallel values from
// the file starting at the current position.
func (oc *obfCodec) ReadParallel() (b *BlockBuffer, err error) {
	return ReadParallel(oc.file, &oc.header)
}

func (oc *obfCodec) ReadSequential() (v [][]float64, ts []int64, err error) {
	return ReadSequential(oc.file, &oc.header)
}

// ----------------------------------------------------------------- //
// Reading Operations -- these operations seek and do validation,
// so are more user-facing and safer
// ----------------------------------------------------------------- //

// Return the last header that had been read. Notice
// header is read upon instantiation.
func (oc *obfCodec) Header() *ObfHeader {
	return &oc.header
}

// Read the entire set of parallel values from the file.
func (oc *obfCodec) Parallel() (b *BlockBuffer, err error) {
	if err = oc.SeekHeader(); err != nil {
		return
	}
	if err = oc.ReadHeader(); err != nil {
		return
	}
	if err = oc.SeekParallel(); err != nil {
		return
	}
	return oc.ReadParallel()
}

// Read the entire set of sequential values from the file.
func (oc *obfCodec) Sequential() (v [][]float64, ts []int64, err error) {
	if err = oc.SeekHeader(); err != nil {
		return
	}
	if err = oc.ReadHeader(); err != nil {
		return
	}
	if err = oc.SeekSequential(); err != nil {
		return
	}
	return oc.ReadSequential()
}

// ----------------------------------------------------------------- //
// Writing Operations -- All these operations happen in-place
// ----------------------------------------------------------------- //

// Write a new header to this file.
func (oc *obfCodec) WriteHeader(h *ObfHeader) (err error) {
	return oc.write(h)
}

// Writes a data frame in parallel mode, assuming the writer
// is at the correct location for the frame.
func (oc *obfCodec) WriteParallel(b *BlockBuffer, tsTransform func(int64) uint32) (err error) {
	return WriteParallel(oc.file, b, tsTransform)
}

func (oc *obfCodec) WriteSequential(b *BlockBuffer, indexFunc func(int64) uint32) (err error) {
	return WriteSequential(oc.file, b, indexFunc)
}
