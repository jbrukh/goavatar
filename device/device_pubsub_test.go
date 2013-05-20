package device

import (
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

	df := &dataFrame{}
	ps.publish(df)

	if one := <-out1; one != df {
		t.Errorf("failed to publish")
	}

	if two := <-out2; two != df {
		t.Errorf("failed to publish")
	}
}
