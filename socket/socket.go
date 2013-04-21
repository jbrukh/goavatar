package socket

import (
	"code.google.com/p/go.net/websocket"
	//. "github.com/jbrukh/goavatar"
	"log"
	"net/http"
)

var engaged = false

func Handler() http.Handler {
	return websocket.Handler(jsonServer)
}

type Control struct {
	State bool
}

type Response struct {
	Channel1 []float64
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
	log.Printf("Avatar Socket Handler is on.")
	for {
		var msg Control
		// Receive receives a text message serialized T as JSON.
		log.Printf("receiving...")
		err := websocket.JSON.Receive(ws, &msg)
		if err != nil {
			log.Println(err)
			break
		}
		log.Printf("recv:%#v\n", msg)
		log.Printf("done")

		// Send send a text message serialized T as JSON.
		r := mockResponse()
		err = websocket.JSON.Send(ws, r)
		if err != nil {
			log.Println(err)
			break
		}
		log.Printf("send:%#v\n", r)
	}
}
