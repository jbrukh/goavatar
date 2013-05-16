//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package formats

import (
	"bytes"
	. "github.com/jbrukh/goavatar"
	"io"
	"log"
	"os"
	"path/filepath"
)

// Recorder that records the Octopus format.
// Warning: calling Start() twice may have unintended
// consequences.
type OBFRecorder struct {
	repo     string    // repository where file is being recorded to
	fileName string    // name of the file/resource id
	file     *os.File  // the file we're writing
	codec    *obfCodec // codec for the OBF format

	out  chan DataFrame // channel for the worker to process frames
	cerr chan error     // channel for worker error feedback

	// diagnostics
	channels   int
	samples    int
	sampleRate int
	buf        *bytes.Buffer
}

func NewOBFRecorder(repo string) *OBFRecorder {
	return &OBFRecorder{
		out:  make(chan DataFrame, DataFrameBufferSize),
		cerr: make(chan error, 1),
		repo: repo,
	}
}

func (r *OBFRecorder) Start() (err error) {
	// buffer the data in this buffer
	r.buf = new(bytes.Buffer)
	go worker(r)
	return
}

func worker(r *OBFRecorder) {
	defer close(r.cerr)
	var (
		tsFirst     int64
		tsTransform = func(ts int64) uint32 {
			return toTs32Diff(ts, tsFirst)
		}
	)
	for {
		// get the frame or die
		df, ok := <-r.out
		if !ok {
			return
		}

		if tsFirst == 0 {
			tsFirst = df.Buffer().Timestamps()[0]
		}

		//log.Printf("writing frame: %v", df)
		// write the frame, or send back an error
		if err := WriteParallelTo(r.buf, df.Buffer(), tsTransform); err != nil {
			r.cerr <- err
			return
		}
	}
}

// Process each incoming frame, if there is an error
func (r *OBFRecorder) ProcessFrame(df DataFrame) error {
	select {
	case err := <-r.cerr:
		close(r.out)
		r.RollbackFile()
		log.Printf("error while processing frame for recording: %v", err)
		return err
	default:
		r.out <- df
		r.channels = df.Buffer().Channels() // TODO
		r.sampleRate = df.SampleRate()      // TODO
		r.samples += df.Buffer().Samples()
	}
	return nil
}

func (r *OBFRecorder) Stop() (id string, err error) {

	// close the worker
	select {
	case <-r.out:
	default:
		close(r.out) // close the worker
	}

	// at this point, the worker may still be operating
	// on the file, therefore we should make sure the worker
	// is done
	err = <-r.cerr
	if err != nil {
		return "", err
	}

	// at this point, the worker is surely exited, so it
	// safe to read from the buffer

	defer func() {
		log.Printf("OBFRecorder: closing the file: %v", r.fileName)
		// TODO: if err, rollback
		r.file.Close()
	}()

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
	if _, err = io.Copy(r.file, r.buf); err != nil {
		return "", err
	}

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
