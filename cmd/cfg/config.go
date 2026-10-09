package cfg

import (
	"github.com/caarlos0/env/v11"
)

type config struct {
	Deploy              bool   `env:"DEPLOY,required,notEmpty"`
	HostDB              string `env:"HOST_DB" envDefault:"localhost"`
	HostValkey          string `env:"HOST_VALKEY" envDefault:"localhost"`
	PortValkey          string `env:"PORT_VALKEY" envDefault:"6379"`
	JwtSecret           string `env:"JWT_SECRET,required,notEmpty"`
	ExpiredRefreshToken int64  `env:"EXPIRED_REFRESH_TOKEN" envDefault:"14"` // в днях
	ExpiredAccessToken  int64  `env:"EXPIRED_ACCESS_TOKEN" envDefault:"15"`  // в минутах
	Domain              string `env:"DOMAIN" default:"test.com"`

	// S3
	Endpoint string `env:"ENDPOINT,required,notEmpty"`
	Region   string `env:"REGION,required,notEmpty"`
	// global Для сервера чисто, не для пользователей (хранение item и т.д., без доступа)
	GlobalAccessKey string `env:"GLOBAL_ACCESS_KEY,required,notEmpty"`
	GlobalSecretKey string `env:"GLOBAL_SECRET_KEY,required,notEmpty"`
	GlobalBucket    string `env:"GLOBAL_BUCKET,required,notEmpty"`
	// photos
	PhotosAucAccessKey string `env:"PHOTOS_ACCESS_KEY,required,notEmpty"`
	PhotosSecretKey    string `env:"PHOTOS_SECRET_KEY,required,notEmpty"`
	PhotosBucket       string `env:"PHOTOS_BUCKET,required,notEmpty"`
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
