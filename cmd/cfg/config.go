package cfg

import (
	"github.com/caarlos0/env/v11"
)

type config struct {
	Deploy              bool   `env:"DEPLOY,required,notEmpty"`
	JwtSecret           string `env:"JWT_SECRET,required,notEmpty"`
	ExpiredRefreshToken int64  `env:"EXPIRED_REFRESH_TOKEN" envDefault:"14"` // в днях
	ExpiredAccessToken  int64  `env:"EXPIRED_ACCESS_TOKEN" envDefault:"15"`  // в минутах
	Domain              string `env:"DOMAIN" default:"test.com"`
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
