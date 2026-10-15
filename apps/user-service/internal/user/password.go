package user

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id cost, the OWASP minimum (19 MiB, 2 passes, 1 lane): one hash
// takes tens of milliseconds and fits the 0.5 GiB a Go container gets.
// The parameters are stored in every hash, so raising them later only
// affects new hashes.
const (
	argonMemoryKiB = 19 * 1024
	argonTime      = 2
	argonThreads   = 1
	argonSaltLen   = 16
	argonKeyLen    = 32
)

// errMalformedHash means a stored hash is not in the format hashPassword writes.
var errMalformedHash = errors.New("malformed password hash")

var b64 = base64.RawStdEncoding

// hashPassword returns an argon2id hash in the PHC string format:
// $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
func hashPassword(password string) (string, error) {

	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemoryKiB, argonTime, argonThreads,
		b64.EncodeToString(salt), b64.EncodeToString(key),
	), nil
}

// verifyPassword reports whether password matches a hash written by
// hashPassword, using the parameters stored in the hash.
func verifyPassword(password, encoded string) (bool, error) {

	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errMalformedHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errMalformedHash
	}

	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, errMalformedHash
	}

	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, errMalformedHash
	}

	want, err := b64.DecodeString(parts[5])
	if err != nil {
		return false, errMalformedHash
	}

	got := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(want)))

	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
