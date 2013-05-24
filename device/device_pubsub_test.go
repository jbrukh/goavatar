//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	. "github.com/jbrukh/goavatar/datastruct"
	"testing"
)

func TestPubSub__New(t *testing.T) {
	if NewPubSub() == nil || NewPubSub().subs == nil {
		t.Errorf("could not instantiate")
	}
}

func TestPubSub__NewSubscriptions(t *testing.T) {
	ps := NewPubSub()
	if len(ps.subs) != 0 {
		t.Errorf("wrong size")
	}

	ps.Subscribe("test1")
	ps.Subscribe("test2")
	if len(ps.subs) != 2 {
		t.Errorf("wrong size")
	}

	ps.Unsubscribe("test1")
	if len(ps.subs) != 1 {
		t.Errorf("wrong size")
	}

	ps.Unsubscribe("test2")
	if len(ps.subs) != 0 {
		t.Errorf("wrong size")
	}
}

func TestPubSub__NewDouble(t *testing.T) {
	ps := NewPubSub()
	if len(ps.subs) != 0 {
		t.Errorf("wrong size")
	}

	ps.Subscribe("test1")
	_, err := ps.Subscribe("test1")
	if err == nil {
		t.Errorf("should have failed")
	}

	if len(ps.subs) != 1 {
		t.Errorf("wrong size")
	}

	ps.Unsubscribe("test1")
	if len(ps.subs) != 0 {
		t.Errorf("wrong size")
	}
}

func TestPubSub__Publish(t *testing.T) {
	ps := NewPubSub()

	out1, err := ps.Subscribe("1")
	if err != nil {
		t.Errorf("could not subscribe")
	}

	out2, err := ps.Subscribe("2")
	if err != nil {
		t.Errorf("could not subscribe")
	}

	df := &MockFrame{}
	ps.publish(df)

	if one := <-out1; one != df {
		t.Errorf("failed to publish")
	}

	if two := <-out2; two != df {
		t.Errorf("failed to publish")
	}

	// now unsubscribe from one
	ps.Unsubscribe("2")
	ps.publish(df)

	if one := <-out1; one != df {
		t.Errorf("failed to publish")
	}
	if _, ok := <-out2; ok {
		t.Errorf("failed to close channel")
	}
}

func TestPubSub__UnsubscribeAll(t *testing.T) {
	ps := NewPubSub()
	var outs [6]chan DataFrame
	outs[0], _ = ps.Subscribe("1")
	outs[1], _ = ps.Subscribe("2")
	outs[2], _ = ps.Subscribe("3")
	outs[3], _ = ps.Subscribe("4")
	outs[4], _ = ps.Subscribe("5")
	outs[5], _ = ps.Subscribe("6")
	if len(ps.subs) != 6 {
		t.Errorf("wrong size")
	}
	ps.UnsubscribeAll()
	if len(ps.subs) != 0 {
		t.Errorf("wrong size")
	}
	for _, v := range outs {
		ensureClosed(t, v)
	}
}
