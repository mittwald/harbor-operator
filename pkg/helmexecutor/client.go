// Package helmexecutor provides methods for executing helm commands as methods.
// It is a wrapper around the helmclient of https://github.com/mittwald/go-helm-client.
package helmexecutor

import (
	"errors"

	helmclient "github.com/mittwald/go-helm-client"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/rest"
)

// HelmExecutor provides an interface for implementing different helmclient types.
type HelmExecutor interface {
	// UpdateHelmRepos updates the list of chart repositories stored in the executors cache.
	UpdateHelmRepos() error

	// AddOrUpdateChartRepo adds a new or updates an existing chart repository.
	AddOrUpdateChartRepo(repo.Entry) error

	// InstallOrUpgradeHelmChart installs or upgrades a chart referenced by helmChart.
	InstallOrUpgradeHelmChart(*helmclient.ChartSpec) error

	// UninstallHelmRelease uninstalls the helmchart referenced by helmChart.
	UninstallHelmRelease(*helmclient.ChartSpec) error
}

// HelmClientExecutor is a wrapper around github.com/mittwald/go-helm-client helmclient
// implementing the HelmExecutor interface.
type HelmClientExecutor struct {
	RepositoryCache string
	RepositoryConfig string
	RestConfig *rest.Config
}

// UpdateHelmRepos implements the HelmExecutor interface.
func (h *HelmClientExecutor) UpdateHelmRepos() error {
	helmClient, err := helmclient.New(&helmclient.Options{
		RepositoryCache:  h.RepositoryCache,
		RepositoryConfig: h.RepositoryConfig,
	})
	if err != nil {
		return err
	}

	return helmClient.UpdateChartRepos()
}

// AddOrUpdateChartRepo implements the HelmExecutor interface.
func (h *HelmClientExecutor) AddOrUpdateChartRepo(chartRepo repo.Entry) error {
	helmClient, err := helmclient.New(&helmclient.Options{
		RepositoryCache:  h.RepositoryCache,
		RepositoryConfig: h.RepositoryConfig,
	})
	if err != nil {
		return err
	}

	return helmClient.AddOrUpdateChartRepo(chartRepo)
}

// InstallOrUpgradeHelmChart implements the HelmExecutor interface.
func (h *HelmClientExecutor) InstallOrUpgradeHelmChart(helmChart *helmclient.ChartSpec) error {
	if h.RestConfig == nil {
		return errors.New("no rest config provided")
	}
	restClientOpts := helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        helmChart.Namespace,
			RepositoryCache:  h.RepositoryCache,
			RepositoryConfig: h.RepositoryConfig,
		},
		RestConfig: h.RestConfig,
	}

	helmClient, err := helmclient.NewClientFromRestConf(&restClientOpts)

	if err != nil {
		return err
	}

	return helmClient.InstallOrUpgradeChart(helmChart)
}

// UninstallHelmRelease implements the HelmExecutor interface.
func (h *HelmClientExecutor) UninstallHelmRelease(helmChart *helmclient.ChartSpec) error {
	if h.RestConfig == nil {
		return errors.New("no rest config provided")
	}

	restClientOpts := helmclient.RestConfClientOptions{
		Options: &helmclient.Options{
			Namespace:        helmChart.Namespace,
			RepositoryCache:  h.RepositoryCache,
			RepositoryConfig: h.RepositoryConfig,
		},
		RestConfig: h.RestConfig,
	}

	helmClient, err := helmclient.NewClientFromRestConf(&restClientOpts)

	if err != nil {
		return err
	}

	return helmClient.UninstallRelease(helmChart)
}
