package env

import (
	"os"
)

const (
	// 環境変数名
	EnvKeyAWSRegion   = "AWS_REGION"
	EnvKeyAWSProfile  = "AWS_PROFILE"
	EnvKeyAWSEndpoint = "AWS_ENDPOINT"
	EnvKeyPort        = "PB_PORT"
	EnvKeyIPAddress   = "PB_IP_ADDRESS"
)

type PBConfig struct {
	AWSRegion   string
	AWSProfile  string
	AWSEndpoint string
	Port        string
	IPAddress   string
}

func LoadPBConfig() *PBConfig {
	pbConfig := &PBConfig{
		AWSRegion:   os.Getenv("AWS_REGION"),
		AWSProfile:  os.Getenv("AWS_PROFILE"),
		AWSEndpoint: os.Getenv("AWS_ENDPOINT"),
		Port:        os.Getenv("PB_PORT"),
		IPAddress:   os.Getenv("PB_IP_ADDRESS"),
	}

	return pbConfig
}
