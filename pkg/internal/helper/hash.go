package helper

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type InterfaceHash []byte

func (hash *InterfaceHash) String() string {
	return fmt.Sprintf("%x", *hash)
}

func (hash *InterfaceHash) Short() string {
	return hash.String()[:8]
}

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
	hash.Write(hashSrc)

	return hash.Sum(nil), nil
}
