package etc

import (
	. "github.com/jbrukh/goavatar"
	. "github.com/jbrukh/goavatar/formats"
	"os"
)

func MockDataFrames(fn string) (d []DataFrame, err error) {
	file, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	codec := NewOBFCodec(file)
	b, err := codec.Parallel()
	if err != nil {
		return
	}

	for b.Samples() > 0 {
		bb := b.DownSample(16)
		df := NewGenericDataFrame(bb, 250)
		d = append(d, df)
	}

	return d, nil
}
