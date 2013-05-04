package formats

import (
	. "github.com/jbrukh/goavatar/devices/avatar"
	//"log"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

const testRepo = "../var"

func TestWriteAndHeader(t *testing.T) {
	r := NewOBFRecorder(testRepo)
	err := r.Start()
	if err != nil || r.fileName == "" || r.file == nil {
		t.Errorf("error starting: %v", err)
	}

	frame := MockAvatarFrames[0]
	err = r.ProcessFrame(frame)
	if err != nil {
		t.Errorf("error processing frame: %v", err)
	}

	id, err := r.Stop()
	if err != nil {
		t.Errorf("error stopping: %v", err)
	}

	// now test the written file
	file, err := os.OpenFile(filepath.Join(testRepo, id), os.O_RDONLY, 0655)
	if err != nil {
		t.Errorf("couldn't open written file: %v", id)
	}
	defer file.Close()

	// decode the header
	codec := &OBFCodec{file: file}
	var header *OBFHeader
	if header, err = codec.ReadHeader(); err != nil {
		t.Error("could not read header")
	}

	// check the header is set
	if codec.Header() == nil {
		t.Errorf("forgot to set the header?")
	}

	// sanity check the header
	if header.Channels != 2 || header.DataType != DataTypeRaw || header.FormatVersion != FormatVersion1 || header.Samples != 16 {
		t.Errorf("header is shot: %v", header)
	}

	err = codec.SeekValues()
	if err != nil {
		t.Errorf("could not seek values: %v", err)
	}

	var val float64
	v := frame.Buffer().ParallelData(0)[0]
	binary.Read(file, binary.BigEndian, &val)
	if val != v {
		t.Errorf("values don't match; expected %d but got %d", v, val)
	}
}
