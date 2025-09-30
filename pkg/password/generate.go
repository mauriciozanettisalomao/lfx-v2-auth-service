package password

import (
	"crypto/rand"
	"math/big"

	"golang.org/x/crypto/bcrypt"
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
