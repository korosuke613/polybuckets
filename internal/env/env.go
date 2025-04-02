package env

import (
	"os"
	"time"
)

const (
	// 環境変数名
	EnvKeyAWSRegion   = "AWS_REGION"
	EnvKeyAWSProfile  = "AWS_PROFILE"
	EnvKeyAWSEndpoint = "AWS_ENDPOINT"
	EnvKeyPort        = "PB_PORT"
	EnvKeyIPAddress   = "PB_IP_ADDRESS"
	EnvKeySiteName    = "PB_SITE_NAME"
)

// PBConfigType holds the configuration values loaded from environment variables.
type PBConfigType struct {
	AWSRegion     string
	AWSProfile    string
	AWSEndpoint   string
	Port          string
	IPAddress     string
	CacheDuration time.Duration
	SiteName      string
}

// LoadPBConfig loads the configuration from environment variables.
func loadPBConfig() *PBConfigType {
	pbConfig := &PBConfigType{
		AWSRegion:   os.Getenv("AWS_REGION"),
		AWSProfile:  os.Getenv("AWS_PROFILE"),
		AWSEndpoint: os.Getenv("AWS_ENDPOINT"),
		Port:        os.Getenv("PB_PORT"),
		IPAddress:   os.Getenv("PB_IP_ADDRESS"),
		SiteName:    os.Getenv("PB_SITE_NAME"),
	}

	// デフォルトのサイト名を設定
	if pbConfig.SiteName == "" {
		pbConfig.SiteName = "polybuckets"
	}

	if os.Getenv("PB_CACHE_DURATION") != "" {
		duration, err := time.ParseDuration(os.Getenv("CACHE_DURATION"))
		if err == nil {
			pbConfig.CacheDuration = duration
		}
	} else {
		pbConfig.CacheDuration = 60 * time.Minute
	}

	// Set UTC as the default timezone
	time.Local = time.UTC

	return pbConfig
}

// PBConfig holds the configuration values loaded from environment variables.
var PBConfig = loadPBConfig()
