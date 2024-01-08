package controller

import (
	"github.com/joeshaw/envdecode"
	"github.com/joho/godotenv"
)

type Config struct {
	GrpcPort      uint16 `env:"GRPC_PORT,default=9000"`
	HttpPort      uint16 `env:"HTTP_PORT,default=443"`
	DiscoveryPort uint16 `env:"DISCOVERY_PORT,default=5050"`
	DbPath        string `env:"DB_PATH,required"`
	AutoCert      struct {
		Enabled  bool   `env:"AUTOCERT_ENABLED,default=false"`
		Domain   string `env:"AUTOCERT_DOMAIN"`
		CacheDir string `enc:"AUTOCERT_CACHEDIR"`
	}
}

func GetConfig() (*Config, error) {
	config := new(Config)
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	err = envdecode.StrictDecode(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
