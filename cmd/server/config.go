package main

import (
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type config struct {
	Server struct {
		Addr                     string        `yaml:"addr"`
		RootURL                  string        `yaml:"rootURL"`
		GracefulTimeout          time.Duration `yaml:"graceFulShutdownTimeout"`
		IndexFileName            string        `yaml:"indexFileName"`
		AssetsFolder             string        `yaml:"assetsFolder"`
		AvatarFolder             string        `yaml:"avatarFolder"`
		MediaFolder              string        `yaml:"mediaFolder"`
		CertFile                 string        `yaml:"certFile"`
		KeyFile                  string        `yaml:"keyFile"`
		JWTSecret                string        `yaml:"jwtSecret"`
		AllowedRequestsPerMinute int           `yaml:"allowedRequestsPerMinute"`
		Webpages                 []string      `yaml:"webpages"`
	} `yaml:"server"`
	Database struct {
		Addr     string `yaml:"addr"`
		Name     string `yaml:"name"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"database"`
	Mailing struct {
		Server   string `yaml:"server"`
		Port     uint16 `yaml:"port"`
		Password string `yaml:"password"`
		User     string `yaml:"user"`
		MailAddr string `yaml:"email"`
	} `yaml:"mailing"`
	InMemoryDB struct {
		Password string `yaml:"password"`
		Addr     string `yaml:"addr"`
	} `yaml:"inmemorydb"`
	UserService struct {
		AllowSignUp              bool          `yaml:"allowSignup"`
		LockOutTimeMinutes       time.Duration `yaml:"lockoutTimeout"`
		NLoginAttempts           int           `yaml:"allowedLoginAttempts"`
		PasswordResetTimeMinutes time.Duration `yaml:"passwordResetTimeout"`
	} `yaml:"userService"`
}

func readConfigFile(configPath string, cfg *config) error {
	f, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	return decoder.Decode(cfg)
}
