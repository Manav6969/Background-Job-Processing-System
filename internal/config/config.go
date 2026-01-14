package config


type Config struct {
	JWTSecret string
	Port      string
	RateLimit int
	
}

func Load() *Config {
	return &Config{
		JWTSecret: "dev_secret_change_later",
		Port:      ":8080",
		RateLimit: 5,
	}
}
