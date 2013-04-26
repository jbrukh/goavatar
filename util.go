package goavatar

// determine whether a trigger switch has been
// flipped (where flipping means to send any
// value over the channel)
func shouldBreak(offSignal <-chan bool) bool {
	select {
	case <-offSignal:
		return true
	default:
	}
	return false
}

// return the average of an array
func averageFloat64(arr []float64) float64 {
	if len(arr) < 1 {
		return float64(0)
	}
	return sumFloat64(arr) / float64(len(arr))
}

// return the sum of an array
func sumFloat64(arr []float64) (result float64) {
	for _, v := range arr {
		result += v
	}
	return
}

func sumInt64(arr []int64) (result int64) {
	for _, v := range arr {
		result += v
	}
	return
}

// return the average of an array
func averageInt64(arr []int64) int64 {
	if len(arr) < 1 {
		return int64(0)
	}
	return int64(float64(sumInt64(arr)) / float64(len(arr)))
}
