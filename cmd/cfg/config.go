package cfg

import (
	"github.com/caarlos0/env/v11"
)

type config struct {
	Deploy           bool   `env:"DEPLOY,required,notEmpty"`
	JwtAccessSecret  string `env:"JWT_ACCESS_SECRET,required,notEmpty"`
	JstRefreshSecret string `env:"JWT_REFRESH_SECRET,required,notEmpty"`
	ExpiredToken     int64  `env:"EXPIRED_TOKEN,required,notEmpty"`
}

var Cfg *config

func Init() error {
	cfg, err := env.ParseAs[config]()
	if err != nil {
		return err
	}
	Cfg = &cfg
	return nil
}
