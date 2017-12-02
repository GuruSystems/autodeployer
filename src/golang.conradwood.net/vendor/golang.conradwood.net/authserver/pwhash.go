package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

/*
func Equal(mac1, mac2 []byte) bool
func New(h func() hash.Hash, key []byte) hash.Hash
*/

// given a challenge and a password will return a hash-string
func PWCrypt(chall string, pw string) (string, error) {
	key := []byte(chall)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(pw))
	b := h.Sum(nil)
	encoded := base64.StdEncoding.EncodeToString([]byte(b))
	return encoded, nil
}
