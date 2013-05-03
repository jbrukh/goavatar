package goavatar

import (
	"errors"
	"io"
	"log"
	"os"
)

// ----------------------------------------------------------------- //
// Constants
// ----------------------------------------------------------------- //

const (
	DataBufferSize   = 1024
	DiagnosticFrames = 10
)

var BadCrcErr = errors.New("frame had bad crc")

// ----------------------------------------------------------------- //
// AvatarEEG Device
// ----------------------------------------------------------------- //

type AvatarDevice struct {
	baseDevice
	serialPort string // serial port like /dev/tty.AvatarEEG03009-SPPDev
}

// NewAvatarDevice creates a new AvatarEEG connection. The user 
// can then start streaming data by calling Connect() and reading the 
// output channel.
func NewAvatarDevice(serialPort string) *AvatarDevice {
	var (
		reader io.ReadCloser
	)

	// connect to the avatar by connecting to the
	// specified serial port
	connFunc := func() (err error) {
		reader, err = os.Open(serialPort)
		return
	}

	// disconnect from the device
	disconnFunc := func() error {
		return reader.Close()
	}

	// the streaming function
	streamFunc := func(c *Control) error {
		return parseByteStream(reader, c)
	}

	recorderProvider := func(token string) Recorder {
		return NewFileRecorder(token)
	}

	return &AvatarDevice{
		baseDevice: *newBaseDevice("AvatarEEG", connFunc, disconnFunc, streamFunc, recorderProvider),
		serialPort: serialPort,
	}
}

// parseByteStream parses the byte stream coming out of the device and writes the output
// to the output channel parameter. It also listens on the offSignal channel for any
// data, in which case it will stop listening the device and return.
func parseByteStream(r io.ReadCloser, c *Control) (err error) {
	parser := NewAvatarParser(r)
	defer c.Close()
	log.Printf("calibrating...")
	// calibrate the device
	frames := make([]*DataFrame, DiagnosticFrames)
	for i := range frames {
		if c.ShouldTerminate() {
			return nil
		}
		log.Printf("frame...")

		// collect the frames for calibration
		frame, err := parser.ParseFrame()
		if err != nil {
			if err == BadCrcErr {
				continue // just skip bad frames
			} else {
				log.Printf("could not calibrate the device: %v", err)
				return err
			}
		}
		frames[i] = frame
	}

	// calibrate -- find the average difference between received time
	// and generated time on the frame
	timeDiff := phase(frames)
	log.Printf("average time diff (ns): %d", timeDiff)

	for {
		if c.ShouldTerminate() {
			return nil
		}

		frame, err := parser.ParseFrame()
		if err != nil {
			log.Printf("error parsing frame: %v", err)
			if err == io.EOF || err == io.ErrUnexpectedEOF || err == io.ErrClosedPipe {
				return err // stream is hosed
			} else {
				continue
			}
		}

		c.Send(frame)
	}
	return nil
}

func phase(frames []*DataFrame) (avg int64) {
	diffs := make([]int64, len(frames))
	for inx, f := range frames {
		diffs[inx] = f.Received().UnixNano() - f.DataFrameHeader.Time().UnixNano()
	}
	avg = averageInt64(diffs)
	log.Printf("time diffs (avg: %d): %v", avg, diffs)
	return
}
