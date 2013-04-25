goavatar
========

Go parser and websocket server for the AvatarEEG data stream.

Installation
============

To get the repo:

    $ go get -u github.com/jbrukh/goavatar

Then, to install the server:

    $ go install github.com/jbrukh/goavatar/server

Make sure the $GOPATH/bin directory is in your PATH and

    $ server

Options
=======

$ server --help
Usage of server:
  -listenPort=8000: the websocket port on which to listen
  -mockDevice=false: whether to use the mock device
  -serialPort="/dev/tty.AvatarEEG03009-SPPDev": the serial port for the device

Protocol
========

        // ControlMessage is the structure that WebSocket clients
        // use to engage and disengage the device. The client
        // will send JSON version of these messages.
        
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
        	Success   bool   `json:"success"`   // whether or not the control message was successful
        	Err       string `json:"err"`       // error text, if any
        	Channels  int    `json:"channels"`  // number of channels
        	VoltRange int    `json:"voltRange"` // range of mVpp measurement of the device
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
