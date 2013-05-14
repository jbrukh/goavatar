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

// -------------------------------------------------------
// Octopus Binary Format (OBF) Version 1 (Parallel Only)
//
// Header (10 bytes):
//    DataType (1 byte):                 0x01 = raw device data;
//    FormatVersion (1 byte):            0x01 = version 1
//    StorageMode (1 byte):              0x01 = parallel; 0x02 = sequential
//    Channels (1 byte):                 0-255 channels
//    Samples (uint32):                   number of samples stored
//    SampleRate (uint16):                 the sample rate at which this data was sampled
//
// Payload (variable):
//    Values + Timestamps
//    (float64*channels*samples
//    + int64*samples):                  parallel format; blocks of channel values + timestamps
//
// -------------------------------------------------------
// Octopus Binary Format (OBF) Version 2 (Combined, 32-bit relative timestamps)
//
// Header (31 bytes):
//    DataType (1 byte):                 0x01 = raw device data;
//    FormatVersion (1 byte):            0x01 = version 1
//    StorageMode (1 byte):              0x01 = parallel; 0x02 = sequential; 0x03 = combined
//    Channels (1 byte):                 0-255 channels
//    Samples (uint32):                   number of samples stored
//    SampleRate (uint16):                 the sample rate at which this data was sampled
//    Endianness (1 byte):               0x00 = Big; 0x01 = Little
//    Reserved (20 bytes):               reserved for future expansions
//
// P-mode Values (variable):
//    Values + Timestamps
//    (float64*channels*samples
//    + uint32*samples):                  parallel format; blocks of channel values +
//                                        timestamps (in ms starting at 0)
//
// S-mode Values (variable):
//    Values (float64*channels*samples):  sequential format
//    Timestamps (uint32*samples):        timestamps of the values (unsigned, in ms starting at 0)
//
// --------------------------------------------------------
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

// ----------------------------------------------------------------- //
// TYPES
// ----------------------------------------------------------------- //

type (
	// OBFCodec will read and write the OBF
	// format on various levels of abstraction.
	OBFCodec struct {
		file   io.ReadWriteSeeker
		header OBFHeader
	}

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
)

func NewOBFCodec(file io.ReadWriteSeeker) *OBFCodec {
	return &OBFCodec{
		file: file,
	}
}

// Return the header, if it has been read. If not,
// the nil header will be returned.
func (s *OBFCodec) Header() *OBFHeader {
	return &s.header
}

// Go to the starting position of the header.
func (s *OBFCodec) SeekHeader() (err error) {
	_, err = s.file.Seek(OBFHeaderAddr, os.SEEK_SET)
	return
}

// Go to the starting position of the values.
func (s *OBFCodec) SeekValues() (err error) {
	_, err = s.file.Seek(int64(OBFValuesAddr), os.SEEK_SET)
	return
}

// In StorageMode parallel, we can seek to individual
// samples.
func (s *OBFCodec) SeekSample(n int) (err error) {
	if s.header.StorageMode != StorageModeParallel {
		return fmt.Errorf("can only seek samples in parallel mode")
	}
	if n > int(s.header.Samples)-1 || n < 0 {
		return fmt.Errorf("no such sample")
	}

	blockSize := int(s.header.Channels)*OBFValueSize + OBFTimestampSize
	offset := int64(OBFHeaderSize + blockSize*n)
	_, err = s.file.Seek(offset, os.SEEK_SET)
	return
}

// Write a new header to this file.
func (s *OBFCodec) WriteHeader(h *OBFHeader) (err error) {
	// go to the start of the file
	if err = s.SeekHeader(); err != nil {
		return err
	}

	err = binary.Write(s.file, binary.BigEndian, h)
	return
}

// Read the OBFHeader of this file.
func (s *OBFCodec) ReadHeader() (header *OBFHeader, err error) {
	if err = s.SeekHeader(); err != nil {
		return nil, err
	}

	err = binary.Read(s.file, binary.BigEndian, &s.header)
	if err != nil {
		return nil, err
	}

	return &s.header, nil
}

// Writes a data frame in parallel mode, assuming the writer
// is at the correct location for the frame.
func (s *OBFCodec) WriteParallel(b *BlockBuffer, firstTs int64) (err error) {
	var (
		samples = b.Samples()
	)

	buf := new(bytes.Buffer)
	for i := 0; i < samples; i++ {
		v, ts := b.NextSample()
		binary.Write(buf, binary.BigEndian, v)
		binary.Write(buf, binary.BigEndian, uint32((ts-firstTs)/1000000))
	}

	//log.Printf("writing parallel blocks: %v", buf.Bytes())
	err = binary.Write(s.file, binary.BigEndian, buf.Bytes())
	//log.Printf("finished: %v", err)
	return
}

func (s *OBFCodec) ReadParallelBlock() (values []float64, ts uint32, err error) {
	if s.header.StorageMode != StorageModeParallel {
		return nil, 0, fmt.Errorf("can only seek samples in parallel mode")
	}
	ch := int(s.header.Channels)
	values = make([]float64, ch)

	err = binary.Read(s.file, binary.BigEndian, values)
	if err != nil {
		return
	}

	err = binary.Read(s.file, binary.BigEndian, &ts)
	return
}

// Read the entire set of parallel values from the file.
func (codec *OBFCodec) Parallel() (b *BlockBuffer, err error) {
	header, err := codec.ReadHeader()
	if err != nil {
		return nil, err
	}

	if err = codec.SeekValues(); err != nil {
		return
	}

	channels, samples := int(header.Channels), int(header.Samples)
	b = NewBlockBuffer(channels, samples)
	v := make([]float64, channels)
	var ts uint32
	for s := 0; s < samples; s++ {
		codec.readBlock(v, &ts)
		b.AppendSample(v, int64(ts))
	}
	return
}

func (s *OBFCodec) readBlock(v []float64, ts *uint32) (err error) {
	if err = binary.Read(s.file, binary.BigEndian, v); err != nil {
		return
	}
	if err = binary.Read(s.file, binary.BigEndian, ts); err != nil {
		return
	}
	return nil
}

// // Convert the entire file into a DataFrame.
// func (s *OBFCodec) ReadDataFrame() (df *GenericDataFrame, err error) {
// 	switch s.header.StorageMode {
// 	case StorageModeParallel:
// 		var (
// 			ch         = int(s.header.Channels)
// 			samples    = int(s.header.Samples)
// 			sampleRate = int(s.header.SampleRate)
// 			buf        = NewBlockBuffer(ch, samples)
// 			timestamps = make([]uint32, samples)
// 		)

// 		// for each sample
// 		for i := 0; i < samples; i++ {
// 			if err = s.SeekSample(i); err != nil {
// 				return nil, err
// 			}
// 			values, ts, err := s.ReadParallelBlock()
// 			if err != nil {
// 				return nil, err
// 			}
// 			buf.PushSlice(values)
// 			timestamps[i] = ts
// 		}

// 		df = NewGenericDataFrame(buf, ch, samples, sampleRate, timestamps)
// 		return
// 	case StorageModeSequential:
// 	default:
// 		return nil, fmt.Errorf("unknown storage mode")
// 	}
// 	return
// }
