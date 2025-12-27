// Package util provides general utility functions for the GoBDS proxy.
package util

import (
	"crypto/md5"
	"encoding/base64"

	"github.com/tailscale/hujson"
)

// ParseCommentedJSON ...
func ParseCommentedJSON(b []byte) ([]byte, error) {
	ast, err := hujson.Parse(b)
	if err != nil {
		return b, err
	}
	ast.Standardize()
	return ast.Pack(), nil
}

// EncryptMessage ...
func EncryptMessage(message, key string) (string, error) {
	hasher := md5.New()
	hasher.Write([]byte(key))
	keyBytes := hasher.Sum(nil)

	messageBytes := []byte(message)

	encryptedBytes := make([]byte, len(messageBytes))
	for i := range messageBytes {
		keyIndex := i % len(keyBytes)
		encryptedBytes[i] = messageBytes[i] ^ keyBytes[keyIndex]
	}

	encoded := base64.URLEncoding.EncodeToString(encryptedBytes)
	return encoded, nil
}
