package config

import "github.com/spf13/viper"

const (
	FlagMetricsAddress          string = "metrics-addr"
	FlagEnableLeaderElection    string = "enable-leader-election"
	FlagHelmClientRepoCachePath string = "helm-client-repo-cache-path"
	FlagHelmClientRepoConfPath  string = "helm-client-repo-conf-path"
)

var (
	MetricsAddr             string
	HelmClientRepoCachePath string
	HelmClientRepoConfPath  string
	EnableLeaderElection    bool
	Config                  config
)

func FromViper() {
	Config.HelmClientRepositoryCachePath = viper.GetString("helm-client-repo-cache-path")
	Config.HelmClientRepositoryConfigPath = viper.GetString("helm-client-repo-conf-path")
	Config.MetricsAddr = viper.GetString("metrics-addr")
	Config.EnableLeaderElection = viper.GetBool("enable-leader-election")
}
