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
func average(arr []float64) float64 {
	if len(arr) < 1 {
		return float64(0)
	}
	return sum(arr) / float64(len(arr))
}

// return the sum of an array
func sum(arr []float64) (result float64) {
	for _, v := range arr {
		result += v
	}
	return
}
