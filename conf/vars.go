package conf

type GlobalConfig struct {
	MODE string `yaml:"mode" mapstructure:"mode"`
	Port string `yaml:"port" mapstructure:"port"` // grpc和http服务监听端口
	Log  struct {
		LogPath string `yaml:"logPath" mapstructure:"logPath"`
		CLS     struct {
			Endpoint    string `yaml:"endpoint" mapstructure:"endpoint"`
			AccessKey   string `yaml:"accessKey" mapstructure:"accessKey"`
			AccessToken string `yaml:"accessToken" mapstructure:"accessToken"`
			TopicID     string `yaml:"topicID" mapstructure:"topicID"`
		} `yaml:"cls" mapstructure:"cls"`
	} `yaml:"log" mapstructure:"log"`
	SentryDsn string `yaml:"sentryDsn" mapstructure:"sentryDsn"`
}
