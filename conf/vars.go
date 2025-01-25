package conf

type GlobalConfig struct {
	MODE string `yaml:"mode"`
	Port string `yaml:"port"` // grpc和http服务监听端口
	Log  struct {
		LogPath string `yaml:"logPath"`
		CLS     struct {
			Endpoint    string `yaml:"endpoint"`
			AccessKey   string `yaml:"accessKey"`
			AccessToken string `yaml:"accessToken"`
			TopicID     string `yaml:"topicID"`
		} `yaml:"cls"`
	} `yaml:"log"`
	SentryDsn string `yaml:"sentryDsn"`
}
