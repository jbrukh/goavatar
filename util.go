//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package goavatar

import (
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
