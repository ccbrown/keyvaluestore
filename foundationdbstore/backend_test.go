package foundationdbstore

import (
	"os"
	"testing"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/subspace"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/keyvaluestore"
	"github.com/ccbrown/keyvaluestore/keyvaluestoretest"
)

func TestBackend(t *testing.T) {
	var db fdb.Database
	var ss subspace.Subspace

	if subspaceStr := os.Getenv("FDB_SUBSPACE"); subspaceStr == "" {
		t.Skip("no foundationdb subspace specified")
	} else {
		fdb.MustAPIVersion(620)
		var err error
		db, err = fdb.OpenDefault()
		require.NoError(t, err)
		ss = subspace.FromBytes([]byte(subspaceStr))
	}

	keyvaluestoretest.TestBackend(t, func() keyvaluestore.Backend {
		_, err := db.Transact(func(tx fdb.Transaction) (interface{}, error) {
			tx.ClearRange(ss)
			return nil, nil
		})
		require.NoError(t, err)

		return &Backend{
			Database: db,
			Subspace: ss,
		}
	})
}
