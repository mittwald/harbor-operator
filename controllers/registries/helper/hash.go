package helper

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	helmclient "github.com/mittwald/go-helm-client"
)

// String returns the string value of an interface hash.
func (hash *InterfaceHash) String() string {
	return fmt.Sprintf("%x", *hash)
}

// Short returns the first 8 characters of an interface hash.
func (hash *InterfaceHash) Short() string {
	return hash.String()[:8]
}

// GenerateHashFromInterfaces returns a hash sum based on a slice of given interfaces.
func GenerateHashFromInterfaces(interfaces []interface{}) (InterfaceHash, error) {
	hashSrc := make([]byte, len(interfaces))

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

// CreateSpecHash returns a hash string constructed with the helm chart spec.
func CreateSpecHash(spec *helmclient.ChartSpec) (string, error) {
	hashSrc, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	toHash := []interface{}{hashSrc}

	hash, err := GenerateHashFromInterfaces(toHash)
	if err != nil {
		return "", err
	}

	return hash.String(), nil
}
