package dlcache

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
)

func hashID(id string) (string, error) {
	h := sha1.New()
	_, err := io.WriteString(h, id)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
