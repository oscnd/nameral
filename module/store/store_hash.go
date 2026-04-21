package store

import (
	"crypto/sha256"
	"encoding/binary"

	"go.scnd.dev/open/polygon/utility/conv"
)

func HashRecord(name, typ, value string) string {
	h := sha256.Sum256([]byte(name + typ + value))
	return conv.Base62(binary.BigEndian.Uint64(h[:8]))
}
