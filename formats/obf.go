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

//
// Octopus Binary Format (OBF)
//
// Header (10 bytes):
//    DataType (1 byte):        		 0x01 = raw device data;
//    FormatVersion (1 byte):   		 0x01 = version 1
//    StorageMode (1 byte):     		 0x01 = parallel; 0x02 = sequential
//    Channels (1 byte):        		 0-255 channels
//    Samples (int32):          		 number of samples stored
//    SampleRate (int16):				 the sample rate at which this data was sampled
//    Values (float64*channels*samples): values in either parallel or sequential format
//    Timestamps (int64*samples):        timestamps of the values
//
// Define v(j,t) to mean the value of channel j (0 < j <= C) at
// sample t (0 <= t < S) where C is the number of channels and
// S is the number of samples. Define T(t) to mean the timestamp
// at time T.
//
// Then "parallel" mode is:
//
//    concat[v(1,t), ..., v(C,t), T(t)] for all t.
//
// For "sequential" mode:
//
//    concat[v(j,0), ..., v(j,S-1)] for all j, followed by
//    [T(t)] for all t.
//

// DataTypes
const (
	DataTypeRaw = 0x01
)

// FormatVersions
const (
	FormatVersion1 = 0x01
)

// StorageModes
const (
	StorageModeParallel   = 0x01
	StorageModeSequential = 0x02
)

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
	}

	OBFParallelBlock struct {
		Values    []float64
		Timestamp int64
	}
)

// Size of the header.
const OBFHeaderSize = 10

// Fixed locations
const (
	OBFHeaderAddr = 0
	OBFValuesAddr = OBFHeaderSize
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

	blockSize := int(s.header.Channels + 1)
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
func (s *OBFCodec) WriteParallelFrame(df DataFrame) (err error) {
	var (
		samples = df.Samples()
		ts      = df.Timestamps()
	)

	buf := new(bytes.Buffer)
	for i := 0; i < samples; i++ {
		binary.Write(buf, binary.BigEndian, df.Buffer().ParallelData(i))
		binary.Write(buf, binary.BigEndian, ts[i])
	}

	//log.Printf("writing parallel blocks: %v", buf.Bytes())
	err = binary.Write(s.file, binary.BigEndian, buf.Bytes())
	//log.Printf("finished: %v", err)
	return
}

func (s *OBFCodec) ReadParallelBlock() (values []float64, ts int64, err error) {
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

func (s *OBFCodec) ReadDataFrame() (b *SamplingBuffer, err error) {
	// TODO
	return
}
