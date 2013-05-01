package goavatar

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var hash = sha1.New()
var repo = "var/" // TODO: generalize

// A recorder of dataframes.
type Recorder interface {
	Start() error
	ProcessFrame(*DataFrame) error
	Stop() (fileName string, err error)
	Token() string
}

type FileRecorder struct {
	file     *os.File
	m        io.Writer
	tempFile string
	token    string
}

// Create a new file recorder with a token.
func NewFileRecorder(token string) *FileRecorder {
	return &FileRecorder{
		token: token,
	}
}

func (r *FileRecorder) Start() (err error) {
	r.tempFile = tmpFile()
	log.Printf("opening file for writing: %v", r.tempFile)
	r.file, err = os.OpenFile(r.tempFile, os.O_CREATE|os.O_WRONLY, 0655)
	if err != nil {
		return err
	}

	// clear the hash
	hash.Reset()
	r.m = io.MultiWriter(hash, r.file)
	return
}

func (r *FileRecorder) ProcessFrame(df *DataFrame) (err error) {
	data := df.Buffer().data
	//l := len(data)

	// TODO: do multiple writes
	err = binary.Write(r.m, binary.BigEndian, data)
	if err != nil {
		return err
	}
	return
}

func (r *FileRecorder) Stop() (fileName string, err error) {
	// open the file
	log.Printf("closing the file: %v", r.tempFile)
	r.file.Close()

	fileName = filepath.Join(repo, fmt.Sprintf("%x.parallel", hash.Sum(nil)))
	if err = os.Rename(r.tempFile, fileName); err != nil {
		log.Printf("couldn't rename temp file: %v", r.tempFile)
	}

	return
}

func (r *FileRecorder) Token() string {
	return r.token
}

func tmpFile() string {
	f := fmt.Sprintf("%s.parallel", time.Now().Format(time.RFC3339Nano))
	return filepath.Join(repo, f)
}
