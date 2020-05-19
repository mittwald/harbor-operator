package internal

import (
	helmclient "github.com/mittwald/go-helm-client"
)

// HelmClientFactory represent functions to dynamically generate helm clients.
type HelmClientFactory func(repoCache, repoConfig, namespace string)(*helmclient.Client, error)
