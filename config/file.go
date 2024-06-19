package config

import (
	"fmt"
	"os"
	"os/user"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const configFileName = ".ghbackup.yaml"

type BackupMethod struct {
	Name          string            `yaml:"name"`
	Enabled       bool              `yaml:"enabled"`
	Type          string            `yaml:"type"`
	TokenInConfig bool              `yaml:"credentialsInConfig"`
	Token         string            `yaml:"token"`
	Parameters    map[string]string `yaml:"parameters"`
}

type fileConfig struct {
	BackupPrivateRepos      bool           `yaml:"backupPrivateRepos"`
	BackupOtherOwnersRepos  bool           `yaml:"backupOtherOwnersRepos"`
	ConcurrentRepoDownloads int            `yaml:"concurrentRepoDownloads"`
	LocalStoragePath        string         `yaml:"localStoragePath"`
	ForceRedownload         bool           `yaml:"forceRedownload"`
	BackupMethods           []BackupMethod `yaml:"backupMethods"`
}

func LoadConfig() {
	usr, err := user.Current()
	if err != nil {
		log.WithError(err).Fatal("failed to get current user")
	}

	configPath := fmt.Sprintf("%s/%s", usr.HomeDir, configFileName)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		createDefaultConfig(configPath)
		return
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.WithError(err).Fatal("failed to read config file")
	}

	var cfg fileConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.WithError(err).Fatal("failed to parse config file")
	}

	// Set the global variables to the values from the config file
	BackupPrivateRepos = cfg.BackupPrivateRepos
	BackupOtherOwnersRepos = cfg.BackupOtherOwnersRepos
	ConcurrentRepoDownloads = cfg.ConcurrentRepoDownloads
	LocalStoragePath = cfg.LocalStoragePath
	ForceRedownload = cfg.ForceRedownload
}

func createDefaultConfig(path string) {
	cfg := fileConfig{
		BackupPrivateRepos:      BackupPrivateRepos,
		BackupOtherOwnersRepos:  BackupOtherOwnersRepos,
		ConcurrentRepoDownloads: ConcurrentRepoDownloads,
		LocalStoragePath:        LocalStoragePath,
		ForceRedownload:         ForceRedownload,
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		log.WithError(err).Fatal("failed to marshal default config")
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		log.WithError(err).WithField("configPath", path).Fatal("failed to write default config to file")
	}

}
