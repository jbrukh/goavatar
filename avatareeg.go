package goavatar

import (
	"errors"
	"io"
	"log"
	"os"
	"time"
)

// ----------------------------------------------------------------- //
// Constants
// ----------------------------------------------------------------- //

const (
	DataBufferSize   = 1024
	DiagnosticFrames = 50
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
	streamFunc := func(offSignal <-chan bool, out chan<- *DataFrame) {
		parseByteStream(reader, offSignal, out)
	}

	return &AvatarDevice{
		baseDevice: *newBaseDevice("AvatarEEG", connFunc, disconnFunc, streamFunc),
		serialPort: serialPort,
	}
}

// parseByteStream parses the byte stream coming out of the device and writes the output
// to the output channel parameter. It also listens on the offSignal channel for any
// data, in which case it will stop listening the device and return.
func parseByteStream(r io.ReadCloser, offSignal <-chan bool, output chan<- *DataFrame) {
	reader := newAvatarParser(r)

	log.Printf("calibrating...")
	// calibrate the device
	frames := make([]*DataFrame, DiagnosticFrames)
	for i, _ := range frames {
		if shouldBreak(offSignal) {
			return
		}

		// collect the frames for calibration
		frame, err := parseFrame(reader)
		if err != nil {
			if err == BadCrcErr {
				continue // just skip bad frames
			} else {
				log.Printf("could not calibrate the device: %v", err)
				return
			}
		}
		frames[i] = frame
	}

	// calibrate -- find the average difference between received time
	// and generated time on the frame
	timeDiff := phase(frames)
	log.Printf("average time diff (ns): %d", timeDiff)

	for {
		// break the loop if 
		// there is an off signal
		if shouldBreak(offSignal) {
			return
		}

		frame, err := parseFrame(reader)
		if err != nil {
			log.Printf("error parsing frame: %v", err)
			if err == io.EOF || err == io.ErrUnexpectedEOF || err == io.ErrClosedPipe {
				return // stream is hosed
			} else {
				continue
			}
		}

		output <- frame
	}
}

func parseFrame(reader *avatarParser) (frame *DataFrame, err error) {
	// read the frame
	err = reader.ConsumeSync()
	if err != nil {
		return
	}

	// once the sync byte has been read,
	// this is technically the time the 
	// frame has been received, assuming
	// it is a correct frame
	t := time.Now()

	header, err := reader.ConsumeHeader()
	if err != nil {
		return
	}

	data, err := reader.ConsumePayload(header)
	if err != nil {
		return
	}

	crc, err := reader.ConsumeCrc()
	if err != nil {
		return
	}

	// collect the frame
	frame = &DataFrame{
		DataFrameHeader: *header,
		data:            data,
		crc:             crc,
		received:        t,
	}
	ourCrc := reader.Crc()
	if ourCrc != crc {
		log.Printf("Bad CRC: %+v (expected: %d)", *frame, ourCrc)
		err = BadCrcErr
	}
	return
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
