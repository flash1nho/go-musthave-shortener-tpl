package helpers

import (
		"crypto/sha256"
    "encoding/base64"
)

func GenerateShortURL(value string) string {
    h := sha256.New()
    h.Write([]byte(value))
    bs := h.Sum(nil)
    hash := base64.URLEncoding.EncodeToString(bs[:])

    return hash[:8]
}
