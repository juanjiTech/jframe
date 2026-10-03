package e

import "errors"

var (
	ErrItemNotFound = errors.New("item not found")
	ErrInvalidName  = errors.New("item name is required")
)
