package main

import (
	"os"
	"gopkg.in/yaml.v2"
)

type config struct {
    Server struct {
		Addr 			string `yaml:"addr"`
		GracefulTimeout int    `yaml:"graceFulShutdownTimeout"`
		IndexFileName	string `yaml:"indexFileName"`
		AssetsFolder	string `yaml:"assetsFolder"`
		CertFile		string `yaml:"certFile"`
		KeyFile			string `yaml:"keyFile"`
    } `yaml:"server"`
    Database struct {
		Addr	 string `yaml:"addr"`
		Name	 string `yaml:"name"`
        User 	 string `yaml:"user"`
        Password string `yaml:"password"`
	} `yaml:"database"`
	Mailing struct {
		Server 	 string `yaml:"server"`
		Port 	 uint16 `yaml:"port"`
		Password string `yaml:"password"`
		User 	 string `yaml:"user"`
		MailAddr string `yaml:"email"`
	} `yaml:"mailing"`
	InMemoryDB struct {
		Password string `yaml:"password"`
		Addr	 string `yaml:"addr"`
	} `yaml:"inmemorydb"`
	UserService struct {
		lockOutTimeMinutes 		 int `yaml:"lockoutTimeout"`
		nLoginAttempts 			 int `yaml:"allowedLoginAttempts"`
		passwordResetTimeMinutes int `yaml:"passwordResetTimeout"`
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