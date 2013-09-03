//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package recorder

import (
	"bytes"
	. "github.com/jbrukh/goavatar/datastruct"
	. "github.com/jbrukh/goavatar/obf"
	. "github.com/jbrukh/goavatar/repo"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type ObfRecorder struct {
	sync.Mutex
	repo     *Repository // repository where file is being recorded to
	fileName string      // name of the file/resource id
	file     *os.File    // the file we're writing
	codec    ObfCodec    // codec for the OBF format

	// diagnostics
	channels   int
	samples    int
	sampleRate int
	buf        bytes.Buffer
	tsFirst    int64
	tsLast     uint32
	fc         int               // frame count
	params     map[string]string // recording parameters
}

func (r *ObfRecorder) tsTransform(ts int64) uint32 {
	return ToTs32Diff(ts, r.tsFirst)
}

func NewObfRecorder(repo *Repository) *ObfRecorder {
	return &ObfRecorder{
		repo: repo,
	}
}

func (r *ObfRecorder) Init() error {
	r.channels = 0
	r.samples = 0
	r.sampleRate = 0
	r.tsFirst = 0
	r.tsLast = 0
	r.fc = 0
	r.buf = bytes.Buffer{}
	return nil
}

// Process each incoming frame, if there is an error
func (r *ObfRecorder) RecordFrame(df DataFrame) error {
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
	buf := df.Buffer()
	samples := buf.Samples()
	r.samples += samples

	// get the last timestamp
	r.tsLast = r.tsTransform(buf.Timestamps()[samples-1])

	// write the frame, or send back an error
	// we are using synchronization to protect the buffer
	r.Lock()
	defer r.Unlock()
	return WriteParallel(&r.buf, df.Buffer(), r.tsTransform)
}

func (r *ObfRecorder) Stop() (id string, err error) {
	return r.commit()
}

func (r *ObfRecorder) Stats() (ms uint32) {
	return r.tsLast
}

func (r *ObfRecorder) commit() (id string, err error) {
	// get the file name
	// TODO: get rid of the subdir parameter
	_, r.fileName = r.repo.NewResourceId()
	log.Printf("ObfRecorder: opening file for writing: %v", r.fileName)

	// make sure the directory exists
	dir := filepath.Dir(r.fileName)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return
	}

	// open the file
	r.file, err = os.OpenFile(r.fileName, os.O_CREATE|os.O_RDWR, 0655)
	if err != nil {
		return
	}
	defer func() {
		log.Printf("ObfRecorder: closing the file: %v", r.fileName)
		// TODO: if err, rollback
		r.file.Close()
	}()

	// get the codec
	r.codec = NewLiveObfCodec(r.file)

	// write the header
	header := &ObfHeader{
		DataType:      DataTypeRaw,
		FormatVersion: FormatVersion2_1,
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
	var b *BlockBuffer
	b, err = r.codec.Parallel()
	if err != nil {
		return "", err
	}

	if err = r.codec.SeekSequential(); err != nil {
		return "", err
	}

	if err = r.codec.WriteSequential(b, ToTs32); err != nil {
		return "", err
	}

	id = filepath.Base(r.fileName)
	return
}

func (r *ObfRecorder) RollbackFile() {
	fileName := r.file.Name()
	log.Printf("ObfRecorder: rolling back %s due to error", fileName)
	r.file.Close()
	if err := os.Remove(fileName); err != nil {
		log.Printf("ObfRecorder: could not remove the file: %s", fileName)
	}
}
