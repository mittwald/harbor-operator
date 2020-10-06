package config

type config struct {
	HelmClientRepositoryCachePath  string `default:"/tmp/.helmcache" split_words:"true"`
	HelmClientRepositoryConfigPath string `default:"/tmp/.helmrepo" split_words:"true"`
}
