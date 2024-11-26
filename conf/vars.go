package conf

type GlobalConfig struct {
	MODE string `yaml:"Mode"`
	Port string `yaml:"Port"` // grpc和http服务监听端口
	Log  struct {
		LogPath string `yaml:"LogPath"`
		CLS     struct {
			Endpoint    string `yaml:"Endpoint"`
			AccessKey   string `yaml:"AccessKey"`
			AccessToken string `yaml:"AccessToken"`
			TopicID     string `yaml:"TopicID"`
		} `yaml:"CLS"`
	} `yaml:"Log"`
	SentryDsn string `yaml:"SentryDsn"`
}
