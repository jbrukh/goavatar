package goavatar

import (
	"code.google.com/p/go.net/websocket"
	"io"
	"log"
	"encoding/json"
	"net/http"
)

//---------------------------------------------------------//
// Constants
//---------------------------------------------------------//

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
// Messages
//---------------------------------------------------------//

// Base type for messages.
type Message struct {
	CorrelationId string `json:"correlationId"` // should be non-empty
	MessageType   string `json:"messageType"`   // will be one of {"connect", "record"}
}

// ConnectMessage is used to connect to the device
// and begin streaming. A ConnectResponseMessage is 
// sent to indicate success or failure, and data
// immediately begins to flow on the data endpoint.
type ConnectMessage struct {
	Message
	Connect   bool   `json:"connect"`   // boolean to engage or disengage the device
	Frequency string `json:"frequency"` // how many messages to send to the front-end per second
}

// RecordMessage is used to trigger recording on
// a device connection that is engaged. A RecordResponseMessage
// is sent to indicate success (if recording has commenced) or
// failure (if the device is off, or other errors).
type RecordMessage struct {
	Message
	Record bool `json:"record"` // start or stop recording
}

// Base type for response messages.
type ResponseMessage struct {
	Message
	Success bool   `json:"success"` // whether or not the control message was successful
	Err     string `json:"err"`     // error text, if any
}

// ConnectResponseMessage is sent in response to a ConnectMessage.
// The MessageType is set to "connect".
type ConnectResponseMessage struct {
	ResponseMessage
	Channels  int `json:"channels"`  // number of channels
	VoltRange int `json:"voltRange"` // range of mVpp measurement of the device
	// may in the future include information about the device
	// sample rate and frames, etc.
}

// RecordResponseMessage is sent in response to a RecordMessage.
// The MessageType is set to "record".
type RecordResponseMessage struct {
	ResponseMessage
}

// DataMessage returns datapoints from the device across 
// the channels. These data points represent incremental data
// that has not been seen before. The data messages come at a 
// frequency specified in the initial control messages.
type DataMessage struct {
	Data      [8]float64 `json:"data"`      // the data for each channel, only first n relevant, n == # of channels
	Timestamp int64      `json:"timestamp"` // timestamp corresponding to this data sample
}

//---------------------------------------------------------//
// The Socket
//---------------------------------------------------------//

func NewControlSocket(device Device, verbose bool) func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			log.Printf("control: listening for incoming messages")
			var message []byte
			err := websocket.Message.Receive(ws, message)
			if err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF || err = io.ErrClosedPipe {
					log.Printf("connection closed")
				}
			}
		}
	}
}

func NewDataSocket(device Device, verbose bool) func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		defer ws.Close()
	}
}

// // NewSocketListener creates a function that can be used
// // as a WebSocket handler. See also Handler(Device).
// func NewSocketListener(device Device, verbose bool) func(ws *websocket.Conn) {
// 	return func(ws *websocket.Conn) {
// 		defer ws.Close()
// 		for {
// 			log.Printf("listening...")
// 			// listen
// 			var msg ControlMessage
// 			if err := websocket.JSON.Receive(ws, &msg); err != nil {
// 				if err == io.EOF {
// 					log.Printf("connection closed")
// 				} else {
// 					log.Printf("error receiving: %v (closing)", err)
// 				}
// 				break
// 			}
// 			log.Printf("received: %+v", msg)

// 			// disengage?
// 			if !msg.Connect {
// 				log.Printf("disengaging the device")
// 				device.Disconnect()
// 				continue
// 			}

// 			// engage?
// 			if msg.Connect {
// 				// device already connected?
// 				if device.Connected() {
// 					send(ws, &ResponseMessage{
// 						Success: false,
// 						Err:     "device already connected",
// 					})
// 					continue
// 				}

// 				// frequency is weird?
// 				if msg.Frequency < 1 || msg.Frequency > 250 {
// 					send(ws, &ResponseMessage{
// 						Success: false,
// 						Err:     "frequency should be 1 to 250",
// 					})
// 					continue
// 				}

// 				// connect
// 				log.Printf("connecting to the device...")
// 				_, err := device.Connect()
// 				if err != nil {
// 					log.Printf("could not connect: %v", err)
// 					send(ws, &ResponseMessage{
// 						Success: false,
// 						Err:     "could not connect to the device",
// 					})
// 					continue
// 				}

// 				log.Printf("device is connected")
// 				defer device.Disconnect()
// 				go stream(device, ws, &msg, verbose)
// 			}
// 		}

// 	}
// }

// // send a message on a the WebSocket
// func send(ws *websocket.Conn, msg interface{}) {
// 	err := websocket.JSON.Send(ws, msg)
// 	if err != nil {
// 		log.Printf("error sending: %s\n", err)
// 	}
// }

// func stream(device Device, ws *websocket.Conn, msg *ControlMessage, verbose bool) {
// 	defer device.Disconnect() // just in case
// 	log.Printf("diagnosing the device...")
// 	// first, diagnose the device
// 	out := device.Out()
// 	var (
// 		channels   int
// 		sampleRate int
// 		batchSize  int
// 	)
// 	if df, ok := <-out; !ok {
// 		log.Printf("device died prematurely, before it could be diagnosed")
// 		return
// 	} else {
// 		// record the diagnostics
// 		channels = df.Channels()
// 		sampleRate, _ = df.SampleRate()

// 		// warning: using "Frequency" here, but really mean
// 		// latency, or period. 1000/L = f. So batch = sampleRate/f = ...
// 		batchSize = sampleRate / msg.Frequency
// 		log.Printf("setting batch size to %d", batchSize)

// 		// send the success response
// 		r := &ResponseMessage{
// 			Success:   true,
// 			Channels:  channels,
// 			VoltRange: df.VoltRange(),
// 		}
// 		log.Printf("sending response: %+v", r)
// 		send(ws, r)
// 	}

// 	buffers := NewMultiBuffer(channels, 1024)

// 	// now run as long as the device is
// 	// connected
// 	for device.Connected() {
// 		df, ok := <-device.Out()
// 		if !ok {
// 			log.Printf("device has disconnected")
// 			return
// 		}

// 		chs := df.ChannelDatas()
// 		//log.Printf("channelDatas: %+v", chs)
// 		buffers.AppendBuffer(chs)

// 		// do we need to keep filling?
// 		if !buffers.HasNext(batchSize) {
// 			continue
// 		}

// 		// ...no, there is enough for a batch
// 		data := new(DataMessage)

// 		for buffers.HasNext(batchSize) {
// 			batch, _ := buffers.Next(batchSize)
// 			log.Printf("appending %d data points", batch.Size())

// 			//log.Printf("batch %v", batch)

// 			for c := 0; c < channels; c++ {
// 				data.Data[c] = batch.data[c][0] // TODO: fix this with getter methods
// 			}

// 			// TODO: fix this
// 			data.Timestamp = df.Time().UnixNano()
// 			if verbose {
// 				log.Printf("sending %+v", data)
// 			}
// 			if err := websocket.JSON.Send(ws, data); err != nil {
// 				log.Printf("error sending: %s\n", err)
// 				return
// 			}
// 		}

// 	}
// }
