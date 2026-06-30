package utils

import (
	"strconv"

	"github.com/marmotedu/errors"
)

func ParseInt64List(strs []string) ([]int64, error) {
	ints := make([]int64, len(strs))
	var err error
	for i, str := range strs {
		ints[i], err = strconv.ParseInt(str, 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert `string` to `int64` (list index: %d)", i)
		}
	}
	return ints, nil
}
