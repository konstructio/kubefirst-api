package taskrunner

import "fmt"

var (
	ErrConflict     = fmt.Errorf("conflicting commands received")
	ErrRootFileCopy = fmt.Errorf("task not supported: copying files at the root level")
)
