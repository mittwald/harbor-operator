package config

import "github.com/spf13/viper"

var Config = &config{
	HelmClientRepositoryCachePath:  viper.GetString(HelmClientRepoCachePath),
	HelmClientRepositoryConfigPath: viper.GetString(HelmClientRepoConfPath),
}

var (
	MetricsAddr             string
	HelmClientRepoCachePath string
	HelmClientRepoConfPath  string
	EnableLeaderElection    bool
)
