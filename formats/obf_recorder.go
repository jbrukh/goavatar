//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package formats

import (
	. "github.com/jbrukh/goavatar"
	"log"
	"os"
	"path/filepath"
)

// Recorder that records the Octopus format.
type OBFRecorder struct {
	repo     string    // repository where file is being recorded to
	fileName string    // name of the file/resource id
	file     *os.File  // the file we're writing
	codec    *OBFCodec // codec for the OBF format

	out  chan DataFrame // channel for the worker to process frames
	cerr chan error     // channel for worker error feedback

	// diagnostics
	channels   int
	samples    int
	sampleRate int
}

func NewOBFRecorder(repo string) *OBFRecorder {
	return &OBFRecorder{
		out:  make(chan DataFrame, DataFrameBufferSize),
		cerr: make(chan error, 1),
		repo: repo,
	}
}

func (r *OBFRecorder) Start() (err error) {
	// get the file name
	r.newFileName()
	log.Printf("OBFRecorder: opening file for writing: %v", r.fileName)

	// open the file
	r.file, err = os.OpenFile(r.fileName, os.O_CREATE|os.O_WRONLY, 0655)
	if err != nil {
		return err
	}

	r.codec = &OBFCodec{file: r.file}

	// make space for the header
	if err = r.codec.SeekValues(); err != nil {
		return
	}

	// open up the worker
	go func() {
		defer close(r.cerr)
		var firstTs int64 = 0
		for {
			// get the frame or die
			df, ok := <-r.out
			if !ok {
				return
			}

			// TODO hacky
			if firstTs == 0 {
				firstTs = df.Generated().UnixNano()
				log.Printf("Seeting first at %d", firstTs)
			}

			//log.Printf("writing frame: %v", df)
			// write the frame, or send back an error
			if err := r.codec.WriteParallel(df.Buffer(), firstTs); err != nil {
				r.cerr <- err
				return
			}
		}
	}()

	return
}

// Process each incoming frame, if there is an error
func (r *OBFRecorder) ProcessFrame(df DataFrame) error {
	select {
	case err := <-r.cerr:
		close(r.out)
		r.RollbackFile()
		return err
	default:
		r.out <- df
		r.channels = df.Channels()     // TODO
		r.sampleRate = df.SampleRate() // TODO
		r.samples += df.Samples()
	}
	return nil
}

func (r *OBFRecorder) Stop() (id string, err error) {

	// close the worker
	close(r.out)

	// at this point, the worker may still be operating
	// on the file, therefore we should make sure the worker
	// is done
	err = <-r.cerr
	if err != nil {
		return "", err
	}

	defer func() {
		log.Printf("OBFRecorder: closing the file: %v", r.fileName)
		r.file.Close()
	}()

	// write the header
	header := &OBFHeader{
		DataType:      DataTypeRaw,
		FormatVersion: FormatVersion2,
		StorageMode:   StorageModeParallel,
		Channels:      uint8(r.channels),
		Samples:       uint32(r.samples),
		SampleRate:    uint16(r.sampleRate),
	}

	if err = r.codec.WriteHeader(header); err != nil {
		return "", err
	}
	return filepath.Base(r.fileName), nil
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
	for {
		f, _ := Uuid()
		r.fileName = filepath.Join(r.repo, f)

		// check for clash just in case
		_, err := os.Stat(r.fileName)
		if err == nil {
			log.Printf("WARNING: new filename clashed with existing: %s", r.fileName)
			continue
		}
		break
	}

}
