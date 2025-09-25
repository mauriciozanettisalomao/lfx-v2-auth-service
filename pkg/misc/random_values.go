// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package misc

import (
	"crypto/rand"
	"math/big"
)

// AlphaNum generates a random alphanumeric string of the specified length
func AlphaNum(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}
