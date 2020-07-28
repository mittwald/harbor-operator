package helper

import (
	"crypto/rand"
)

// NewRandomPassword returns a random secret string with a given length.
func NewRandomPassword(passwordStrength int32) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"

	if passwordStrength == 0 {
		passwordStrength = 8
	}

	buf := make([]byte, passwordStrength)

	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}

	for i, b := range buf {
		buf[i] = letters[b%byte(len(letters))]
	}

	return string(buf), nil
}
