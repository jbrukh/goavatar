//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package util

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// return the average of an array
func AverageFloat64(arr []float64) float64 {
	if len(arr) < 1 {
		return float64(0)
	}
	return SumFloat64(arr) / float64(len(arr))
}

// return the sum of an array
func SumFloat64(arr []float64) (result float64) {
	for _, v := range arr {
		result += v
	}
	return
}

func SumInt64(arr []int64) (result int64) {
	for _, v := range arr {
		result += v
	}
	return
}

// return the average of an array
func AverageInt64(arr []int64) int64 {
	if len(arr) < 1 {
		return int64(0)
	}
	return int64(float64(SumInt64(arr)) / float64(len(arr)))
}

func AbsFloat64(f float64) float64 {
	if f >= 0 {
		return f
	}
	return -f
}

func NanosToTime(nanos int64) time.Time {
	nsec := nanos % 1000000000
	sec := (nanos - nsec) / 1000000000
	return time.Unix(sec, nsec)
}

func Uuid() (uuid string, err error) {
	file, err := os.OpenFile("/dev/urandom", os.O_RDONLY, 0)
	if err != nil {
		return
	}
	defer file.Close()
	b := make([]byte, 16)
	if _, err = file.Read(b); err != nil {
		return
	}
	uuid = fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return
}

// Return the timestamp of the s-th sample given that the duration
// between timestamps is dur.
func InterpolateTs(start int64, s int, δ time.Duration) int64 {
	return start + int64(s)*int64(δ)
}

// For testing panics.
func TestPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r != nil {
			// ok
		}
	}()
	f()
	t.Errorf("should have panicked")
}
