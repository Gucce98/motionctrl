package utils

import (
	"reflect"
	"strconv"
)

func InSlice(val interface{}, array interface{}) (exists bool, index int) {
	exists = false
	index = -1

	switch reflect.TypeOf(array).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(array)

		for i := 0; i < s.Len(); i++ {
			if reflect.DeepEqual(val, s.Index(i).Interface()) == true {
				index = i
				exists = true
				return
			}
		}
	}

	return
}

func BlockSlideSlice(array interface{}, blockSize int, f func(interface{}) bool) {
	run := true
	switch reflect.TypeOf(array).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(array)
		n := s.Len()

		for i := 0; (i < n) && run; i += blockSize {
			if i+blockSize > n { //TODO improve here (?)
				run = f(s.Slice(i, n).Interface())
			} else {
				run = f(s.Slice(i, i+blockSize).Interface())
			}
		}
	}
}

func ToInt64Slice(array []string) ([]int64, error) {
	var err error
	chatids := make([]int64, len(array))
	for i, v := range array {
		chatids[i], err = strconv.ParseInt(v, 10, 64)

		if err != nil {
			return nil, err
		}
	}

	return chatids, nil
}
