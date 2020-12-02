package config

import "github.com/spf13/viper"

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
