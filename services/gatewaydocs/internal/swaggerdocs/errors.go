package swaggerdocs

import "errors"

var (
	ErrNotFoundDocs    = errors.New("not found docs")
	ErrFailedMergeDocs = errors.New("failed to merge docs")
)
