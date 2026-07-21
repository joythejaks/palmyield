package auth

import (
	"crypto/rand"

	"golang.org/x/crypto/bcrypt"
)

const tempPasswordChars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"

// GenerateTempPassword returns a random password for admin-created (invited)
// accounts. The admin relays it to the farmer out-of-band (no email/SMS
// delivery infra exists yet).
func GenerateTempPassword(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i, b := range buf {
		buf[i] = tempPasswordChars[int(b)%len(tempPasswordChars)]
	}
	return string(buf), nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
