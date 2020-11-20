package keyvaluestore

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAtomicWriteConflict(t *testing.T) {
	err := &AtomicWriteConflictError{
		Err: fmt.Errorf("foo"),
	}
	assert.True(t, IsAtomicWriteConflict(err))
}
