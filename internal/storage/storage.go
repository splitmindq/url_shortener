package storage

import "errors"

var (
	ErrUrlNotFound = errors.New("url not found")
	ErrAliasExists = errors.New("alias already exists")
)
