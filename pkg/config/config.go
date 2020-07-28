package config

import "github.com/spf13/viper"

var Config config

func FromViper() {
	Config.HelmClientRepositoryCachePath = viper.GetString("helm-client-repo-cache-path")
	Config.HelmClientRepositoryConfigPath = viper.GetString("helm-client-repo-conf-path")
}
