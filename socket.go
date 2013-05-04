package goavatar

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

//---------------------------------------------------------//
// Constants
//---------------------------------------------------------//

// the kickoff channel, sharing state between
// the control and data endpoints; only one
// connect request can succeed
var (
	kickoff   = make(chan bool, 1)
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
func DataHandler(device Device, verbose bool, integers bool) http.Handler {
	return websocket.Handler(NewDataSocket(device, verbose, integers))
}

//---------------------------------------------------------//
// Messages
//---------------------------------------------------------//

// Base type for messages.
type Message struct {
	Id          string `json:"id"`           // should be non-empty
	MessageType string `json:"message_type"` // will be one of {"info", connect", "record", "error"}
}

// Basic information about the server.
type InfoMessage struct {
	Id          string `json:"id"`           // should be non-empty
	MessageType string `json:"message_type"` // should be "info"
}

// ConnectMessage is used to connect to the device
// and begin streaming. A ConnectResponseMessage is 
// sent to indicate success or failure, and data
// immediately begins to flow on the data endpoint.
type ConnectMessage struct {
	Id          string `json:"id"`           // should be non-empty
	MessageType string `json:"message_type"` // should be "connect"
	Connect     bool   `json:"connect"`      // boolean to engage or disengage the device
	Pps         int    `json:"pps"`          // points per second, one of 250, 125, 83, ..., 250/k
	BatchSize   int    `json:"batch_size"`   // points to return per batch
}

// RecordMessage is used to trigger recording on
// a device connection that is engaged. A RecordResponseMessage
// is sent to indicate success (if recording has commenced) or
// failure (if the device is off, or other errors).
type RecordMessage struct {
	Id          string `json:"id"`           // should be non-empty
	MessageType string `json:"message_type"` // should be "record"
	Record      bool   `json:"record"`       // start or stop recording
	Token       string `json:"token"`        // authentication token for upload
}

// Base type for response messages.
type Response struct {
	Id          string `json:"id"`           // echo of your correlation id
	MessageType string `json:"message_type"` // will be one of {"info", connect", "record", "error"}
	Success     bool   `json:"success"`      // whether or not the control message was successful
	Err         string `json:"err"`          // error text, if any
}

// ConnectResponseMessage is sent in response to a ConnectMessage.
// The MessageType is set to "connect".
type ConnectResponse struct {
	Id          string `json:"id"`           // echo of your correlation id
	MessageType string `json:"message_type"` // will be "connect"
	Success     bool   `json:"success"`      // whether or not the control message was successful
	Err         string `json:"err"`          // error text, if any
	Status      string `json:"status"`       // device status, one of {"armed", "busy", "disconnected"}
}

// RecordResponseMessage is sent in response to a RecordMessage.
// The MessageType is set to "record".
type RecordResponse struct {
	Id          string `json:"id"`           // echo of your correlation id
	MessageType string `json:"message_type"` // will be "record"
	Success     bool   `json:"success"`      // whether or not the control message was successful
	Err         string `json:"err"`          // error text, if any
	File        string `json:"file"`         // output file
}

type InfoResponse struct {
	Id          string `json:"id"`           // echo of your correlation id
	MessageType string `json:"message_type"` // will be "info"
	Success     bool   `json:"success"`      // whether or not the control message was successful
	Err         string `json:"err"`          // error text, if any
	Version     string `json:"version"`      // octopus server version
	DeviceName  string `json:"device_name"`  // device name
}

// DataMessage returns datapoints from the device across 
// the channels. These data points represent incremental data
// that has not been seen before. The data messages come at a 
// frequency specified in the initial control messages.
type DataMessage struct {
	Data      [][]float64 `json:"data"`       // the data for each channel, only first n relevant, n == # of channels
	Ints      [][]int64   `json:"ints"`       // the data for each channel, as integers
	LatencyMs float64     `json:"latency_ms"` // the running latency
	//Timestamp int64      `json:"timestamp"` // timestamp corresponding to this data sample

}

//---------------------------------------------------------//
// The Socket
//---------------------------------------------------------//

func NewControlSocket(device Device, verbose bool) func(ws *websocket.Conn) {
	// return the actual handler function
	return func(conn *websocket.Conn) {
		defer conn.Close()
		controller := &SocketController{
			conn:    conn,
			kickoff: kickoff, // there is only one kickoff channel
			device:  device,
		}

		for {
			log.Printf("listening for incoming messages")

			msgBytes, msgBase, err := controller.Receive()
			if err != nil {
				if err == io.EOF || err.Error() == "EOF" {
					break
				}
				continue
			}

			log.Printf("received: %s", msgBytes)

			// message types
			msgType := msgBase.MessageType
			switch msgType {

			case "info":
				controller.ProcessInfoMessage(msgBytes, msgBase.Id)

			case "connect":
				controller.ProcessConnectMessage(msgBytes, msgBase.Id)

			case "record":
				controller.ProcessRecordMessage(msgBytes, msgBase.Id)

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
	kickoff chan bool
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
		log.Printf("websocket: %v", err)
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
	log.Printf("sending response: %+v", r)
	// send it off
	err := websocket.JSON.Send(s.conn, r)
	if err != nil {
		log.Printf("error sending: %v\n", err)
	}
}

func (s *SocketController) ProcessInfoMessage(msgBytes []byte, id string) {
	log.Printf("INFO")

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
	log.Printf("CONNECT")

	var msg ConnectMessage
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		s.SendErrorResponse(id, err.Error())
	}

	// start building the response
	r := new(ConnectResponse)
	r.MessageType = "connect"
	r.Id = msg.Id

	// should we disconnect?
	if !msg.Connect {
		err := s.device.Disconnect()
		if err != nil {
			r.Success = false
			r.Status = "error"
			r.Err = err.Error()
		} else {
			r.Success = true
			r.Status = "disconnected"
			// also, disarm the device
			select {
			case <-s.kickoff:
			default:
			}
		}
		goto Respond
	}

	// should we connect?
	if msg.Connect {

		// are the parameters sane?
		if msg.Pps < 1 || msg.Pps > 250 {
			r.Success = false
			r.Err = "pps should be between 1 and 250"
			goto Respond
		}

		if msg.BatchSize > msg.Pps {
			r.Success = false
			r.Err = "batchSize should not exceed pps"
			goto Respond
		}

		// maybe someone is already using it
		if s.device.Connected() {
			r.Success = false
			r.Status = "busy"
			r.Err = "device is already connected"
		}

		// ok, now we can tell the data endpoint
		// to stream when it is connected

		select {
		case s.kickoff <- true:
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
			goto Respond

		default:
			// the kickoff channel is blocked, so some
			// other request has armed the device for
			// streaming
			r.Success = false
			r.Status = "armed"
			r.Err = "device is already armed"
			goto Respond
		}

	}

Respond:
	s.SendResponse(r)
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

	if !s.device.Connected() {
		r.Err = "device is not streaming"
		goto Respond
	}

	if !msg.Record {
		outFile, err := s.device.Stop()
		if err == nil {
			r.Success = true
			r.File = outFile
		}
		goto Respond
	}

	if msg.Record {
		if s.device.Recording() {
			r.Err = "already recording"
			goto Respond
		}

		err = s.device.Record(msg.Token)
		if err != nil {
			r.Err = err.Error()
			goto Respond
		}
		r.Success = true
	}

Respond:
	s.SendResponse(r)
}

func NewDataSocket(device Device, verbose bool, integers bool) func(ws *websocket.Conn) {
	return func(conn *websocket.Conn) {
		defer conn.Close()

		// gate to see if it is armed
		select {
		case <-kickoff:
			// we connect and begin to stream
			if !device.Connected() {
				err := device.Connect()
				if err != nil {
					log.Printf("could not connect: %v", err)
				}
				stream(conn, device, verbose, integers)
			} else {
				log.Printf("WARNING: device was already operating")
			}

		default:
			// kickoff is blocked, meaning no one has
			// armed the device; we close the socket
			return
		}
	}
}

func stream(conn *websocket.Conn, device Device, verbose bool, integers bool) {
	out := device.Out()
	defer device.Disconnect()

	// diagnose the situation
	df, ok := <-out
	if !ok {
		return
	}

	// get the channels
	channels := df.Channels()
	devicePps := df.SampleRate()

	// just in case something went wrong
	if pps < 1 || pps > devicePps {
		pps = DefaultPps
		log.Printf("WARNING: setting default PPS")
	}

	if batchSize > pps || batchSize < 1 {
		batchSize = DefaultBatchSize
		log.Printf("WARNING: setting default batchSize")
	}

	// latency calculation
	frames := 0
	mean_diff := float64(0)

	// now we need to sample every devicePps/pps points
	sampleRate := devicePps / pps

	// actual number of data points we must read
	// in order to obtain a sampled batch of batchSize 
	absBatchSize := batchSize * sampleRate

	b := NewSamplingBuffer(channels, sampleRate*batchSize*10, sampleRate)
	kill := make(chan bool)
	for {
		// break if there was an error sending
		// a message over the socket
		select {
		case <-kill:
			return
		default:
		}

		df, ok := <-out
		if !ok {
			return
		}

		// calculate the latency
		frames++
		d := absFloat64(float64(df.Received().UnixNano() - df.Time().UnixNano()/1000000)) // diff between received and stamped time
		mean_diff = float64(frames)/float64(frames+1)*mean_diff + d/float64(frames+1)

		b.Append(df.Buffer())
		for b.Size() > absBatchSize {
			batch := b.SampleNext(absBatchSize)

			// send it off
			go func() {
				msg := new(DataMessage)
				msg.LatencyMs = absFloat64(mean_diff - d)
				if integers {
					msg.Ints = make([][]int64, channels)
					for i, _ := range msg.Ints {
						ch := batch.ChannelData(i)
						msg.Ints[i] = make([]int64, len(ch))
						for j, _ := range msg.Ints[i] {
							msg.Ints[i][j] = int64(ch[j] * float64(Multiplier))
						}
					}
				} else {
					msg.Data = make([][]float64, channels)
					for i, _ := range msg.Data {
						msg.Data[i] = batch.ChannelData(i)
					}
				}
				if verbose {
					log.Printf("sending data msg: %+v", msg)
				}
				err := websocket.JSON.Send(conn, msg)
				if err != nil {
					log.Printf("error sending data msg: %v\n", err)
					kill <- true
				}
			}()
		}
	}
}
