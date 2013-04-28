package goavatar

import (
	"testing"
)

func newEmptyDevice() *baseDevice {
	connFunc := func() error {
		return nil
	}

	disconnFunc := func() error {
		return nil // do nothing
	}

	streamFunc := func(control <-chan ControlCode, output chan<- *DataFrame) {
		select {
		case <-control:
			return
		default:
		}
	}

	return newBaseDevice(
		"UnitTestMockDevice",
		connFunc,
		disconnFunc,
		streamFunc,
	)
}

func TestConnectionLogic(t *testing.T) {
	d := newEmptyDevice()
	d.Connect()
	if !d.Connected() {
		t.Errorf("didn't connect")
	}

	_, err := d.Connect()
	if err == nil {
		t.Errorf("failed to block second connect")
	}

	d.Disconnect()
	if d.Connected() {
		t.Errorf("failed to disconnect")
	}

	d.Disconnect()
	d.Disconnect()
	d.Disconnect()
	d.Disconnect()

	if d.Connected() {
		t.Errorf("connected now for some reason")
	}

	d.Connect()
	if !d.Connected() {
		t.Errorf("didn't connect for a second time")
	}
}

func TestCleanupLogic(t *testing.T) {
	d := newEmptyDevice()
	if d.out != nil {
		t.Errorf("has an out channel for some reason")
	}

	_, err := d.Connect()
	if err != nil {
		t.Errorf("failed to connect")
	}

	if d.out == nil {
		t.Errorf("didn't create out channel")
	}

	err = d.Disconnect()
	if err != nil {
		t.Errorf("failed to disconnect")
	}
	ensureClosed(t, d.out)
}

func ensureClosed(t *testing.T, out chan *DataFrame) {
	defer func() {
		if r := recover(); r != nil {
		}
	}()
	out <- &DataFrame{}
	t.Errorf("failed to panic when writing to this channel, hence it is still open")
}
