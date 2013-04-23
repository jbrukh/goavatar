package socket

import (
	"code.google.com/p/go.net/websocket"
	. "github.com/jbrukh/goavatar"
	"github.com/jbrukh/window"
	"log"
	"net/http"
)

var engaged = false

const (
	MaxFrames      = 10000
	WindowSize     = 1000
	WindowMultiple = 10
)

func Handler() http.Handler {
	return websocket.Handler(jsonServer)
}

type Control struct {
	State bool
}

type Response struct {
	Channel1 []float64
	Channel2 []float64
}

func mockResponse() *Response {
	r := &Response{
		Channel1: make([]float64, 10),
	}
	for i := 0; i < 10; i++ {
		r.Channel1[i] = float64(i)
	}
	return r
}

func jsonServer(ws *websocket.Conn) {
	var device Device
	for {
		var msg Control

		err := websocket.JSON.Receive(ws, &msg)
		if err != nil {
			log.Println(err)
			break
		}

		if msg.State {
			// connect
			log.Println("Connecting to the device...")

			// set up the device
			device := NewMockDevice()

			// connect to it
			out, err := device.Connect()
			if err != nil {
				log.Printf("Error: %v\n", err)
				continue
			}

			var (
				w1 = window.New(WindowSize, WindowMultiple)
				w2 = window.New(WindowSize, WindowMultiple)
			)

			go run(out, w1, w2, ws)
			// do not reference w1,w2 here, not thread-safe

		} else {
			// disconnect
			log.Println("Disconnecting from the device...")
			device.Disconnect()

		}
	}
}

func run(out <-chan *DataFrame, w1, w2 *window.MovingWindow, ws *websocket.Conn) {
	for i := 0; i < MaxFrames; i++ {
		df, ok := <-out
		if !ok {
			log.Printf("The data channel got closed (exiting)")
			return
		}
		//log.Printf("Got df: %v", df.String())
		for _, v := range df.ChannelData(1) {
			w1.PushBack(v)
		}
		for _, v := range df.ChannelData(2) {
			w2.PushBack(v)
		}

		r := &Response{
			Channel1: w1.Slice(),
			Channel2: w2.Slice(),
		}
		err := websocket.JSON.Send(ws, r)
		if err != nil {
			log.Printf("error sending: %s\n", err)
			continue
		}
		log.Printf("send:%#v\n", r)
	}
}
