package qdb

import "errors"

var (
	ErrInvalidArgs = errors.New("invalid arguments")
	ErrNotFound    = errors.New("key not found")
)
