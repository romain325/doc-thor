package services

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrBuildNotRunning = errors.New("build is not in running state")
)
