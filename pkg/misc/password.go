// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package misc

import (
	"golang.org/x/crypto/bcrypt"
)

// GeneratePasswordPair generates a random password and returns both plain text and bcrypt hash
func GeneratePasswordPair(length int) (plainPassword, bcryptHash string, err error) {
	// Generate random 20-character password
	plainPasswordGenerated, errAlphaNum := AlphaNum(length)
	if errAlphaNum != nil {
		return "", "", errAlphaNum
	}
	plainPassword = plainPasswordGenerated

	// Hash with bcrypt (cost 10 is standard)
	hashedPassword, errGenerateFromPassword := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if errGenerateFromPassword != nil {
		return "", "", errGenerateFromPassword
	}
	bcryptHash = string(hashedPassword)

	return plainPassword, bcryptHash, nil
}
