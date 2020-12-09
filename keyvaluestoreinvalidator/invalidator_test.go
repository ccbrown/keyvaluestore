package keyvaluestoreinvalidator_test

import (
	"testing"

	"github.com/ccbrown/keyvaluestore"
	"github.com/ccbrown/keyvaluestore/keyvaluestoreinvalidator"
	"github.com/ccbrown/keyvaluestore/keyvaluestoretest"
	"github.com/ccbrown/keyvaluestore/memorystore"
)

func TestReadCache(t *testing.T) {
	keyvaluestoretest.TestBackend(t, func() keyvaluestore.Backend {
		return &keyvaluestoreinvalidator.Invalidator{
			Backend:    memorystore.NewBackend(),
			Invalidate: func(string) {},
		}
	})
}
