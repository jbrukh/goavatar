package socket

import (
	"code.google.com/p/go.net/websocket"
	. "github.com/jbrukh/goavatar"
	//"github.com/jbrukh/window"
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
	Connect   bool `json:"connect"`   // boolean to engage or disengage the device
	Frequency int  `json:"frequency"` // how many messages to deliver per second
	Average   bool `json:"average"`   // if false, last data point from each batch will be sent; otherwise average of the batch
}

// ResponseMessage is sent by the socket in response to
// a ControlMessage. If an error occurred, then Success
// will be set to false, and the Err will be set to the
// error message.
type ResponseMessage struct {
	Success  bool   `json:"success"`  // whether or not the control message was successful
	Err      string `json:"err"`      // error text, if any
	Channels int    `json:"channels"` // number of channels

	// may in the future include information about the device
	// sample rate and frames, etc.
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

// NewSocketListener creates a function that can be used
// as a WebSocket handler. See also Handler(Device).
func NewSocketListener(device Device) func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			// listen
			var msg ControlMessage
			if err := websocket.JSON.Receive(ws, &msg); err != nil {
				log.Printf("error receiving: %v (closing)", err)
				break
			}
			log.Printf("received: %+v", msg)

			// disengage?
			if !msg.Connect {
				log.Printf("disengaging the device")
				device.Disconnect()
				continue
			}

			// engage?
			if msg.Connect {
				// device already connected?
				if device.Connected() {
					send(ws, &ResponseMessage{
						Success: false,
						Err:     "device already connected",
					})
					continue
				}

				// connect
				log.Printf("connecting to the device...")
				_, err := device.Connect()
				if err != nil {
					log.Printf("could not connect: %v", err)
					send(ws, &ResponseMessage{
						Success: false,
						Err:     "could not connect to the device",
					})
					continue
				}

				log.Printf("device is connected")
				defer device.Disconnect()
				go stream(device, ws, &msg)
			}
		}

	}
}

// send a message on a the WebSocket
func send(ws *websocket.Conn, msg interface{}) {
	err := websocket.JSON.Send(ws, msg)
	if err != nil {
		log.Printf("error sending: %s\n", err)
	}
}

func stream(device Device, ws *websocket.Conn, msg *ControlMessage) {
	defer device.Disconnect() // just in case
	log.Printf("diagnosing the device...")
	// first, diagnose the device
	out := device.Out()
	if df, ok := <-out; !ok {
		log.Printf("device died prematurely, before it could be diagnosed")
		return
	} else {
		// send the success response
		send(ws, &ResponseMessage{
			Success:  true,
			Channels: df.Channels(),
		})
	}

	// now run as long as the device is
	// connected
	for device.Connected() {
		df, ok := <-device.Out()
		if !ok {
			log.Printf("device has closed")
			return
		}

		data := produce(df)
		if err := websocket.JSON.Send(ws, data); err != nil {
			log.Printf("error sending: %s\n", err)
			break
		}
	}
}

func produce(df *DataFrame) *DataMessage {
	data := new(DataMessage)
	channels := df.Channels()
	for i := 0; i < channels; i++ {
		data.Data[i] = df.ChannelData(i + 1)[0] // get the first point
	}
	data.Timestamp = df.Time().UnixNano()
	return data
}
