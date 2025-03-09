package storage

import (
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/trie"
	"strconv"
)

var idDbFp = "./record/ldb/s" + "95" + "/n" + strconv.FormatUint(27, 10)

// NewLevelDBDatabase creates a persistent key-value database without a freezer
// moving immutable chain segments into cold storage.
var diskIdDB, _ = rawdb.NewLevelDBDatabase(idDbFp, 0, 1, "accountState", false)

var IdDB = trie.NewDatabaseWithConfig(diskIdDB, &trie.Config{
	Cache:     0,
	Preimages: true,
})
