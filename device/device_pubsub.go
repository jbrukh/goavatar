//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	"fmt"
	. "github.com/jbrukh/goavatar/datastruct"
	"log"
	"sync"
)

// PubSub is a thread-safe publisher-subscriber. It is
// used by Devices to stream data frames to interested
// parties (streamers and recorders).
type PubSub struct {
	sync.Mutex
	subs map[string]chan DataFrame
}

// Create a new PubSub.
func NewPubSub() *PubSub {
	return &PubSub{
		subs: make(map[string]chan DataFrame),
	}
}

// Subscribe to this PubSub with the given name. The data
// channel will be returned.
func (ps *PubSub) Subscribe(name string) (out chan DataFrame, err error) {
	ps.Lock()
	defer ps.Unlock()
	if _, ok := ps.subs[name]; ok {
		log.Printf("subscription '%s' already exists", name)
		return nil, fmt.Errorf("subscription already exists")
	}
	out = make(chan DataFrame, DataFrameBufferSize)
	ps.subs[name] = out
	return
}

// Unsubscribe the subscriber with the given name from
// this PubSub.
func (ps *PubSub) Unsubscribe(name string) {
	ps.Lock()
	defer ps.Unlock()
	ps.unsubscribe(name)
}

func (ps *PubSub) unsubscribe(name string) {
	if out, ok := ps.subs[name]; ok {
		close(out)
	}
	delete(ps.subs, name)
}

// Unsubscribe all subscribers from this PubSub.
func (ps *PubSub) UnsubscribeAll() {
	ps.Lock()
	defer ps.Unlock()
	for name, out := range ps.subs {
		close(out)
		delete(ps.subs, name)
	}
}

func (ps *PubSub) publish(df DataFrame) {
	ps.Lock()
	defer ps.Unlock()
	for _, v := range ps.subs {
		v <- df
	}
}
