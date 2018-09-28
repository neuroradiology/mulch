package main

import (
	"path"

	"github.com/BurntSushi/toml"
)

// LEProductionString is the magic string to use production LE directory
const LEProductionString = "LETS_ENCRYPT_PRODUCTION"

// AppConfig describes the general configuration of an App
type AppConfig struct {
	// persistent storage
	DataPath string

	// ACME directory server
	AcmeURL string

	// ACME for issued certificate alerts
	AcmeEmail string

	// Listen HTTP address
	HTTPAddress string

	// Listen HTTPS address
	HTTPSAddress string

	// global mulchd configuration path
	configPath string
}

type tomlAppConfig struct {
	DataPath     string `toml:"data_path"`
	AcmeURL      string `toml:"proxy_acme_url"`
	AcmeEmail    string `toml:"proxy_acme_email"`
	HTTPAddress  string `toml:"proxy_listen_http"`
	HTTPSAddress string `toml:"proxy_listen_https"`
}

// NewAppConfigFromTomlFile return a AppConfig using
// mulchd.toml config file in the given configPath
func NewAppConfigFromTomlFile(configPath string) (*AppConfig, error) {

	filename := path.Clean(configPath + "/mulchd.toml")

	appConfig := &AppConfig{
		configPath: configPath,
	}

	// defaults (if not in the file)
	tConfig := &tomlAppConfig{
		DataPath:     "./var/data", // example: /var/lib/mulch
		AcmeURL:      "https://acme-staging.api.letsencrypt.org/directory",
		AcmeEmail:    "root@localhost.localdomain",
		HTTPAddress:  ":80",
		HTTPSAddress: ":443",
	}

	if _, err := toml.DecodeFile(filename, tConfig); err != nil {
		return nil, err
	}

	appConfig.DataPath = tConfig.DataPath

	appConfig.AcmeURL = tConfig.AcmeURL
	if appConfig.AcmeURL == LEProductionString {
		appConfig.AcmeURL = "" // acme package default is production directory
	}
	appConfig.AcmeEmail = tConfig.AcmeEmail
	appConfig.HTTPAddress = tConfig.HTTPAddress
	appConfig.HTTPSAddress = tConfig.HTTPSAddress

	return appConfig, nil
}
