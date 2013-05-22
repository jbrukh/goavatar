//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package formats

import (
	"bytes"
	. "github.com/jbrukh/goavatar/datastruct"
	. "github.com/jbrukh/goavatar/util"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type OBFRecorder struct {
	sync.Mutex
	repo     string    // repository where file is being recorded to
	fileName string    // name of the file/resource id
	file     *os.File  // the file we're writing
	codec    *obfCodec // codec for the OBF format

	// diagnostics
	channels   int
	samples    int
	sampleRate int
	buf        bytes.Buffer
	tsFirst    int64
	fc         int // frame count
}

func (r *OBFRecorder) tsTransform(ts int64) uint32 {
	return toTs32Diff(ts, r.tsFirst)
}

func NewOBFRecorder(repo string) *OBFRecorder {
	return &OBFRecorder{
		repo: repo,
	}
}

func (r *OBFRecorder) Init() error {
	return nil
}

// Process each incoming frame, if there is an error
func (r *OBFRecorder) RecordFrame(df DataFrame) error {
	if df == nil {
		return nil
	}
	r.fc++

	// on the first frame, obtain the first timestamp
	// and normalize to that
	if r.fc == 1 {
		if b := df.Buffer(); b != nil {
			ts := b.Timestamps()
			if len(ts) > 0 {
				r.tsFirst = ts[0]
			}
			r.sampleRate = df.SampleRate()
			r.channels = b.Channels()
		}
	}
	r.samples += df.Buffer().Samples()

	// write the frame, or send back an error
	// we are using synchronization to protect the buffer
	r.Lock()
	defer r.Unlock()
	return WriteParallelTo(&r.buf, df.Buffer(), r.tsTransform)
}

func (r *OBFRecorder) Stop() (id string, err error) {
	return r.commit()
}

func (r *OBFRecorder) commit() (id string, err error) {
	// get the file name
	r.newFileName()
	log.Printf("OBFRecorder: opening file for writing: %v", r.fileName)

	// open the file
	r.file, err = os.OpenFile(r.fileName, os.O_CREATE|os.O_RDWR, 0655)
	if err != nil {
		return
	}
	defer func() {
		log.Printf("OBFRecorder: closing the file: %v", r.fileName)
		// TODO: if err, rollback
		r.file.Close()
	}()

	// get the codec
	r.codec = newObfCodec(r.file)

	// write the header
	header := &OBFHeader{
		DataType:      DataTypeRaw,
		FormatVersion: FormatVersion2,
		StorageMode:   StorageModeCombined,
		Channels:      uint8(r.channels),
		Samples:       uint32(r.samples),
		SampleRate:    uint16(r.sampleRate),
	}
	if err = r.codec.WriteHeader(header); err != nil {
		return "", err
	}

	// write the parallel frames from the buffer
	r.Lock()
	if _, err = io.Copy(r.file, &r.buf); err != nil {
		return "", err
	}
	r.Unlock()

	//read the parallel frames from the buffer as a BlockBuffer
	b, err := r.codec.Parallel()
	if err != nil {
		return "", err
	}

	if err = r.codec.SeekSequential(); err != nil {
		return "", err
	}

	if err = r.codec.WriteSequential(b, toTs32); err != nil {
		return "", err
	}

	id = filepath.Base(r.fileName)
	return
}

func (r *OBFRecorder) RollbackFile() {
	fileName := r.file.Name()
	log.Printf("OBFRecorder: rolling back %s due to error", fileName)
	r.file.Close()
	if err := os.Remove(fileName); err != nil {
		log.Printf("OBFRecorder: could not remove the file: %s", fileName)
	}
}

// return the name of the recording file
func (r *OBFRecorder) newFileName() {
	for i := 0; i < 100; i++ {
		f, _ := Uuid()
		r.fileName = filepath.Join(r.repo, f)

		// check for clash just in case
		_, err := os.Stat(r.fileName)
		if err == nil {
			log.Printf("WARNING: new filename clashed with existing: %s", r.fileName)
			continue
		}
		return
	}
	panic("could not generate filename")
}
