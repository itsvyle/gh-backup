package config

import (
	"fmt"
	"os"
	"os/user"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const configFileName = ".ghbackup.yaml"

type fileConfig struct {
	BackupPrivateRepos      bool   `yaml:"backupPrivateRepos"`
	BackupOtherOwnersRepos  bool   `yaml:"backupOtherOwnersRepos"`
	ConcurrentRepoDownloads int    `yaml:"concurrentRepoDownloads"`
	LocalStoragePath        string `yaml:"localStoragePath"`
	ForceRedownload         bool   `yaml:"forceRedownload"`
}

func LoadConfig() {
	usr, err := user.Current()
	if err != nil {
		log.WithError(err).Fatal("failed to get current user")
	}

	configPath := fmt.Sprintf("%s/%s", usr.HomeDir, configFileName)

	// Check if the config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		createDefaultConfig(configPath)
		return
	}

	// Read the config file contents
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.WithError(err).Fatal("failed to read config file")
	}

	// Unmarshal the YAML data into the Config struct
	var cfg fileConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.WithError(err).Fatal("failed to parse config file")
	}
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
