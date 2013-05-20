//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	"fmt"
	"log"
	"sync"
)

type PubSub struct {
	sync.Mutex
	subs map[string]chan DataFrame
}

func NewPubSub() *PubSub {
	return &PubSub{
		subs: make(map[string]chan DataFrame),
	}
}

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
