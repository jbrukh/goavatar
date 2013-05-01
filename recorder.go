package goavatar

import (
	"encoding/binary"
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
	fileName string
	file     *os.File
}

func NewFileRecorder(fileName string) *FileRecorder {
	return &FileRecorder{
		fileName: fileName,
	}
}

func (r *FileRecorder) Start() (err error) {
	log.Printf("opening file for writing: %v", r.fileName)
	r.file, err = os.OpenFile(r.fileName, os.O_CREATE|os.O_WRONLY, 0655)
	return
}

func (r *FileRecorder) ProcessFrame(df *DataFrame) (err error) {
	data := df.Buffer().data
	//l := len(data)

	// TODO: do multiple writes
	err = binary.Write(r.file, binary.BigEndian, data)
	if err != nil {
		return err
	}
	return
}

func (r *FileRecorder) Stop() (err error) {
	// open the file
	log.Printf("closing the file: %v", r.fileName)
	r.file.Close()
	return
}
