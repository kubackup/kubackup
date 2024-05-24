package config

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Data       Data             `yaml:"data"`
	Logger     LoggerConfig     `yaml:"logger"`
	Jwt        JwtConfig        `yaml:"jwt"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
}

type PrometheusConfig struct {
	Enable bool `yaml:"enable"`
}

type ServerConfig struct {
	Name  string     `yaml:"name"`
	Bind  BindConfig `yaml:"bind"`
	Debug bool       `yaml:"debug"` //是否开启debug模式
}

type JwtConfig struct {
	Key    string `yaml:"key"`    // token私钥
	MaxAge int    `yaml:"maxAge"` // token有效期
}

type BindConfig struct {
	Host string `yaml:"host"` // 服务ip
	Port int    `yaml:"port"` // 服务端口
}

type Data struct {
	NoCache  bool   `yaml:"noCache"`
	CacheDir string `yaml:"cacheDir"` //缓存目录
	DbDir    string `yaml:"dbDir"`    //数据库目录
}

type LoggerConfig struct {
	Level   string `yaml:"level"`   // 日志级别
	LogPath string `yaml:"logPath"` // 日志文件路径
}
