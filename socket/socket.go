package socket

import (
	"code.google.com/p/go.net/websocket"
	. "github.com/jbrukh/goavatar"
	"github.com/jbrukh/window"
	"log"
	"net/http"
)

//---------------------------------------------------------//
// Constants
//---------------------------------------------------------//

const ()

//---------------------------------------------------------//
// Handler -- for use with net/http HTTP server
//---------------------------------------------------------//

// Handler provides a request handler for use with Go's HTTP 
// server. To set the handler, call:
//
//    http.Handle("/uri", socket.Handler(device))
//
func Handler(device Device) http.Handler {
	return websocket.Handler(NewSocketListener(device))
}

//---------------------------------------------------------//
// Messages
//---------------------------------------------------------//

// ControlMessage is the structure that WebSocket clients
// use to engage and disengage the device.
type ControlMessage struct {
	Engage    bool `json:"engage"`    // boolean to engage or disengage the device
	Frequency int  `json:"frequency"` // how many messages to deliver per second
	Average   bool `json:"average"`   // if false, last data point from each batch will be sent; otherwise average of the batch
}

// ResponseMessage is sent by the socket in response to
// a ControlMessage. If an error occurred, then Success
// will be set to false, and the Err will be set to the
// error message.
type ResponseMessage struct {
	Success bool   `json:"success"` // whether or not the control message was successful
	Err     string `json:"err"`     // error text, if any

	// may in the future include information about the device
	// sample rate and frames, etc.
}

// DataMessage returns datapoints from the device across 
// the channels. These data points represent incremental data
// that has not been seen before. The data messages come at a 
// frequency specified in the initial control messages.
type DataMessage struct {
	Channels  int        `json:"channels"`  // number of channels
	Data      [8]float64 `json:"data"`      // the data for each channel, only first n relevant, n == # of channels
	Timestamp uint32     `json:"timestamp"` // timestamp corresponding to this data sample
}

//---------------------------------------------------------//
// The Socket
//---------------------------------------------------------//

// NewSocketListener creates a function that can be used
// as a WebSocket handler. See also Handler(Device).
func NewSocketListener(device Device) func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {

	}
}

func send(ws *websocket.Conn, msg interface{}) {
	err := websocket.JSON.Send(ws, msg)
	if err != nil {
		log.Printf("error sending: %s\n", err)
	}
}

func run(device Device) {
	for i := 0; i < MaxFrames; i++ {
		df, ok := <-device.Out()
		if !ok {
			log.Printf("disconnecting from the device because output channel has closed")
			return
		}

		r := &Response{
			Channel1: df.ChannelData(1),
			Channel2: df.ChannelData(2),
		}
		err := websocket.JSON.Send(ws, r)
		if err != nil {
			log.Printf("error sending: %s\n", err)
			break
		}
		log.Printf("send:%#v\n", r)
	}
	device.Disconnect()

}
