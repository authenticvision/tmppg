package errutil

import "errors"

func IsType[T error](err error) bool {
	var target T
	return errors.As(err, &target)
}
