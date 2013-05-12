//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package socket

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	. "github.com/jbrukh/goavatar"
	"io"
	"log"
	"net/http"
	"path/filepath"
)

//---------------------------------------------------------//
// Constants
//---------------------------------------------------------//

// the kickoff channel, sharing state between
// the control and data endpoints; only one
// connect request can succeed
var (
	kickoff   = make(chan *SocketController, 1)
	pps       int // points per second
	batchSize int // batch size
)

const (
	DefaultPps       = 125
	DefaultBatchSize = 25
)

var Multiplier int64 = 10000000000000000

//---------------------------------------------------------//
// Handlers -- for use with net/http HTTP server
//---------------------------------------------------------//

// ControlHandler provides a request handler for use with Go's HTTP
// server for the control endpoint. To set the handler, call:
//
//    http.Handle("/uri", socket.ControlHandler(device, true))
//
func ControlHandler(device Device, verbose bool) http.Handler {
	return websocket.Handler(NewControlSocket(device, verbose))
}

// DataHandler provides a request handler for use with Go's HTTP
// server for the data endpoint. To set the handler, call:
//
//    http.Handle("/uri", socket.DataHandler(device, true))
//
func DataHandler(device Device, verbose bool) http.Handler {
	return websocket.Handler(NewDataSocket(device, verbose))
}

//---------------------------------------------------------//
// The Socket
//---------------------------------------------------------//

func NewControlSocket(device Device, verbose bool) func(ws *websocket.Conn) {
	// return the actual handler function
	return func(conn *websocket.Conn) {
		defer conn.Close()
		defer device.Disconnect()
		controller := &SocketController{
			conn:    conn,
			kickoff: kickoff, // there is only one kickoff channel
			device:  device,
		}

		for {
			msgBytes, msgBase, err := controller.Receive()
			if err != nil {
				if err == io.EOF || err.Error() == "EOF" {
					break
				}
				continue
			}

			log.Printf("SOCKET: RECEIVED %s", msgBytes)

			// message types
			msgType := msgBase.MessageType
			switch msgType {

			case "info":
				controller.ProcessInfoMessage(msgBytes, msgBase.Id)

			case "connect":
				controller.ProcessConnectMessage(msgBytes, msgBase.Id)

			case "record":
				controller.ProcessRecordMessage(msgBytes, msgBase.Id)

			case "upload":
				controller.ProcessUploadMessage(msgBytes, msgBase.Id)

			default:
				errStr := fmt.Sprintf("unknown message type: '%s'", msgType)
				controller.SendErrorResponse(msgBase.Id, errStr)
				continue
			}
		}
	}
}

// SocketController encapsulates all of the
// business logic of sending and receiving
// control messages.
type SocketController struct {
	conn    *websocket.Conn
	kickoff chan *SocketController
	device  Device
}

// Receive receives control messages. If there is
// a problem with the connection, or the message
// you send has a bad "header", then an err is
// reported.
func (s *SocketController) Receive() (msgBytes []byte, msgBase Message, err error) {
	// get the raw bytes
	err = websocket.Message.Receive(s.conn, &msgBytes)
	if err != nil {
		log.Printf("websocket says: %v", err)
		return
	}

	// get the type
	err = json.Unmarshal(msgBytes, &msgBase)
	if err != nil {
		s.SendErrorResponse(msgBase.Id, "error getting message type")
	}
	return
}

// SendErrorResponse sends error messages; these
// are usually for internal server errors.
func (s *SocketController) SendErrorResponse(id, errStr string) {
	// create the error message
	r := new(Response)
	r.MessageType = "error"
	r.Id = id
	r.Success = false
	r.Err = errStr

	// log
	log.Printf("sending error response: %+v", r)

	// send it off
	err := websocket.JSON.Send(s.conn, r)
	if err != nil {
		log.Printf("error sending: %v\n", err)
	}
	return
}

func (s *SocketController) SendResponse(r interface{}) {
	log.Printf("SOCKET RESPONSE: %+v", r)
	// send it off
	err := websocket.JSON.Send(s.conn, r)
	if err != nil {
		log.Printf("error sending: %v\n", err)
	}
}

func (s *SocketController) ProcessInfoMessage(msgBytes []byte, id string) {
	var msg InfoMessage
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		s.SendErrorResponse(id, err.Error())
	}

	r := new(InfoResponse)
	r.MessageType = "info"
	r.Id = msg.Id
	r.Success = true
	r.Version = "0.1"
	r.DeviceName = s.device.Name()

	s.SendResponse(r)
}

func (s *SocketController) ProcessConnectMessage(msgBytes []byte, id string) {
	var msg ConnectMessage
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		s.SendErrorResponse(id, err.Error())
	}

	// start building the response
	r := new(ConnectResponse)
	r.MessageType = "connect"
	r.Id = msg.Id
	r.Success = false
	defer s.SendResponse(r)

	// should we disconnect?
	if !msg.Connect {
		err := s.device.Disconnect()
		if err != nil {
			r.Status = "error"
			r.Err = err.Error()
		} else {
			r.Success = true
			r.Status = "disarmed"
			// also, disarm the device
			select {
			case <-s.kickoff:
			default:
			}
		}
		return
	}

	// should we connect?
	if msg.Connect {

		// are the parameters sane?
		if msg.Pps < 1 || msg.Pps > 250 {
			r.Err = "pps should be between 1 and 250"
			return
		}

		if msg.BatchSize > msg.Pps {
			r.Err = "batchSize should not exceed pps"
			return
		}

		// maybe someone is already using it
		if s.device.Connected() {
			r.Status = "busy"
			r.Err = "device is already connected"
			return
		}

		// ok, now we can tell the data endpoint
		// to stream when it is connected

		select {
		case s.kickoff <- s: // send self on the kickoff channel
			// set the parameters; WARNING: since the
			// device is already armed at this point, users
			// should wait for our ConnectResponse before
			// attempting to connect to the data endpoint
			pps = msg.Pps
			batchSize = msg.BatchSize

			// device can accept a value, meaning
			// no one request for connection is in
			// an "armed" state, so we have succeeded
			r.Success = true
			r.Status = "armed"
			return

		default:
			// the kickoff channel is blocked, so some
			// other request has armed the device for
			// streaming
			r.Status = "armed"
			r.Err = "device is already armed"
			return
		}
	}
}

func (s *SocketController) ProcessRecordMessage(msgBytes []byte, id string) {
	var msg RecordMessage
	var err error
	if err = json.Unmarshal(msgBytes, &msg); err != nil {
		s.SendErrorResponse(id, err.Error())
	}

	r := new(RecordResponse)
	r.MessageType = "record"
	r.Id = msg.Id
	r.Success = false
	defer s.SendResponse(r)

	if !s.device.Connected() {
		r.Err = "device is not streaming"
		return
	}

	if !msg.Record {
		outFile, err := s.device.Stop()
		if err == nil {
			r.Success = true
			r.ResourceId = outFile
		}
		return
	}

	if msg.Record {
		if s.device.Recording() {
			r.Err = "already recording"
			return
		}

		err = s.device.Record()
		if err != nil {
			r.Err = err.Error()
			return
		}
		r.Success = true
	}
}

func (s *SocketController) ProcessUploadMessage(msgBytes []byte, id string) {
	var msg UploadMessage
	var err error
	if err = json.Unmarshal(msgBytes, &msg); err != nil {
		s.SendErrorResponse(id, err.Error())
	}

	r := new(UploadResponse)
	r.MessageType = "upload"
	r.Id = msg.Id
	r.Success = false
	defer s.SendResponse(r)

	// perform the upload
	var (
		resId    = msg.ResourceId
		token    = msg.Token
		endpoint = msg.Endpoint
		file     = filepath.Join(s.device.Repo(), resId)
	)

	err = UploadOBFFile(file, endpoint, token)
	if err != nil {
		r.Err = err.Error()
		return
	}

}

func NewDataSocket(device Device, verbose bool) func(ws *websocket.Conn) {
	return func(conn *websocket.Conn) {
		defer conn.Close()

		// gate to see if it is armed
		select {
		case controller := <-kickoff:

			msg := new(ConnectResponse)
			msg.MessageType = "connect"
			msg.Success = false

			// we connect and begin to stream
			if !device.Connected() {
				err := device.Connect()
				if err != nil {
					log.Printf("could not connect: %v", err)
					msg.Err = fmt.Sprintf("could not connect to the device")
					msg.Status = "disarmed"
					controller.SendResponse(msg)
					return
				}
				msg.Success = true
				msg.Status = "streaming"
				controller.SendResponse(msg)
				stream(conn, device, verbose)
			} else {
				log.Printf("WARNING: device was already operating")
				return
			}

		default:
			// kickoff is blocked, meaning no one has
			// armed the device; we close the socket
			return
		}
	}
}

func stream(conn *websocket.Conn, device Device, verbose bool) {
	// logging
	log.Printf("DEVICE: STREAMING ON")
	defer log.Printf("DEVICE: STREAMING OFF")

	// device stuff
	defer device.Disconnect()
	out := device.Out()

	// diagnose the situation
	df, ok := <-out
	if !ok {
		return
	}
	var (
		channels   = df.Channels()
		sampleRate = df.SampleRate()
	)

	// check the parameters
	if pps < 1 || pps > sampleRate {
		pps = DefaultPps
		log.Printf("WARNING: setting default PPS")
	}
	if batchSize > pps || batchSize < 1 {
		batchSize = DefaultBatchSize
		log.Printf("WARNING: setting default batchSize")
	}

	streamLoop()
}

func streamLoop() {
	var (
		frames       = 0
		mean_diff    = float64(0)
		kill         = make(chan bool)
		pluckRate    = sampleRate / pps      // now we need to sample every sampleRate/pps points
		absBatchSize = batchSize * pluckRate // actual number of data points we must read in order to obtain a sampled batch of batchSize
		b            = NewBlockBuffer(channels, sampleRate*batchSize*10)

		shouldReturn = func() bool {
			select {
			case <-kill:
				return true
			default:
			}
			return false
		}
	)

	b.PluckRate(pluckRate)

	for {
		// break if there was an error sending
		// a message over the socket
		var (
			df DataFrame
			ok bool
		)

		if df, ok = <-out; !ok || shouldReturn() {
			return
		}

		// calculate the latency
		frames++
		mean_diff = updateMeanDiff(frames, mean_diff, df)

		// put the frame into our memory buffer
		b.Append(df.Buffer())

		// while there are batches, return them
		for b.Size() > absBatchSize {
			if shouldReturn() {
				return
			}

			var (
				batch = b.DownSample(absBatchSize)
				msg   = new(DataMessage)
			)

			msg.LatencyMs = AbsFloat64(mean_diff - d)
			msg.Data = batchToArrays(batch)
			if verbose {
				log.Printf("sending data msg: %+v", msg)
			}
			err := websocket.JSON.Send(conn, msg)
			if err != nil {
				log.Printf("error sending data msg: %v\n", err)
				kill <- true
			}
		}
	}
}

func batchToArrays(batch *BlockBuffer) [][]float64 {
	res := make([][]float64, batch.Channels())
	for i := range res {
		res[i] = make([]float64, batch.Size())
	}
	for s := 0; s < batch.Size(); c++ {
		v, _ := batch.ReadBlock()
		for c, value := range v {
			res[c][s] = value
		}
	}
}

func updateMeanDiff(frames int, mean_diff float64, df DataFrame) float64 {
	d := AbsFloat64(float64(df.Received().UnixNano() - df.Generated().UnixNano())) // diff between received and stamped time
	mean_diff = float64(frames)/float64(frames+1)*mean_diff + d/float64(frames+1)
}
