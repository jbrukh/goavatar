package goavatar

import (
	"log"
	"os"
)

// A recorder of dataframes.
type Recorder interface {
	Start() error
	ProcessFrame(*DataFrame) error
	Stop() error
}

type FileRecorder struct {
	file *os.File
}

func NewFileRecorder(file string) *FileRecorder {
	return &FileRecorder{
	// TODO!
	}
}

func (r *FileRecorder) Start() (err error) {
	// open the file
	log.Printf("opening the file")
	return
}

func (r *FileRecorder) ProcessFrame(df *DataFrame) (err error) {
	// open the file
	log.Printf("recording the frame %#v", *df)
	return
}

func (r *FileRecorder) Stop() (err error) {
	// open the file
	log.Printf("closing the file")
	return
}
