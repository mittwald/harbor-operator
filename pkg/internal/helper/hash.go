package helper

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type InterfaceHash []byte

// String returns the string value of an interface hash
func (hash *InterfaceHash) String() string {
	return fmt.Sprintf("%x", *hash)
}

// Short returns the first 8 characters of an interface hash
func (hash *InterfaceHash) Short() string {
	return hash.String()[:8]
}

// GenerateHashFromInterfaces returns a hash sum based on a slice of given interfaces
func GenerateHashFromInterfaces(interfaces []interface{}) (InterfaceHash, error) {
	var hashSrc []byte

	for _, in := range interfaces {
		chainElem, err := json.Marshal(in)
		if err != nil {
			return InterfaceHash{}, err
		}

		hashSrc = append(hashSrc, chainElem...)
	}

	hash := sha256.New()
	_, err := hash.Write(hashSrc)
	if err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}
