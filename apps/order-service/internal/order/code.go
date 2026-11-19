package order

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// codePrefix names the shop in every order code.
const codePrefix = "MA-"

// codeDigits gives only 10^6 codes, so two orders can draw the same one: the
// caller must check a code is free before using it.
const codeDigits = 6

// codeSpace is 10^codeDigits, the number of possible codes.
var codeSpace = big.NewInt(1_000_000)

// newCode returns a random public order code such as MA-048291. It is also
// the holder of the order's books in the catalogue.
func newCode() string {

	// rand.Reader never fails: it crashes the program instead.
	n, _ := rand.Int(rand.Reader, codeSpace)

	// Leading zeros keep every code the same length.
	return fmt.Sprintf("%s%0*d", codePrefix, codeDigits, n)
}
