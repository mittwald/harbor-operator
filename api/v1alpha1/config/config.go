package config

var Config config

var (
	MetricsAddr             string
	HelmClientRepoCachePath string
	HelmClientRepoConfPath  string
	EnableLeaderElection    bool
)

func FromViper() {
	Config.HelmClientRepositoryCachePath = HelmClientRepoCachePath
	Config.HelmClientRepositoryConfigPath = HelmClientRepoConfPath
}
