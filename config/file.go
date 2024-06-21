package config

import (
	"fmt"
	"os"
	"os/user"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const ConfigFileName = ".ghbackup.yaml"

type BackupMethod struct {
	Name          string            `yaml:"name"`
	Enabled       bool              `yaml:"enabled"`
	Type          string            `yaml:"type"`
	TokenInConfig bool              `yaml:"credentialsInConfig"`
	Token         string            `yaml:"token"`
	Parameters    map[string]string `yaml:"parameters"`
}

type fileConfig struct {
	Debug                     *bool          `yaml:"debug"`
	NonInteractive            *bool          `yaml:"nonInteractive"`
	BackupPrivateRepos        *bool          `yaml:"backupPrivateRepos"`
	BackupOtherOwnersRepos    *bool          `yaml:"backupOtherOwnersRepos"`
	ConcurrentRepoDownloads   *int           `yaml:"concurrentRepoDownloads"`
	ConcurrentBackupUploaders *int           `yaml:"concurrentBackupUploaders"`
	LocalStoragePath          *string        `yaml:"localStoragePath"`
	ForceRedownload           *bool          `yaml:"forceRedownload"`
	DeleteDataAfterUpload     *bool          `yaml:"deleteDataAfterUpload"`
	BackupMethods             []BackupMethod `yaml:"backupMethods"`
}

func LoadConfig() {
	usr, err := user.Current()
	if err != nil {
		log.WithError(err).Fatal("failed to get current user")
	}

	configPath := fmt.Sprintf("%s/%s", usr.HomeDir, ConfigFileName)

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

	if cfg.Debug != nil {
		Debug = *cfg.Debug
	}
	if cfg.NonInteractive != nil {
		NonInteractive = *cfg.NonInteractive
	}
	if cfg.BackupPrivateRepos != nil {
		BackupPrivateRepos = *cfg.BackupPrivateRepos
	}
	if cfg.BackupOtherOwnersRepos != nil {
		BackupOtherOwnersRepos = *cfg.BackupOtherOwnersRepos
	}
	if cfg.ConcurrentRepoDownloads != nil {
		ConcurrentRepoDownloads = *cfg.ConcurrentRepoDownloads
	}
	if cfg.ConcurrentBackupUploaders != nil {
		ConcurrentBackupUploaders = *cfg.ConcurrentBackupUploaders
	}
	if cfg.LocalStoragePath != nil {
		LocalStoragePath = *cfg.LocalStoragePath
	}
	if cfg.ForceRedownload != nil {
		ForceRedownload = *cfg.ForceRedownload
	}
	if cfg.DeleteDataAfterUpload != nil {
		DeleteDataAfterUpload = *cfg.DeleteDataAfterUpload
	}

	BackupMethods = cfg.BackupMethods
}

func createDefaultConfig(path string) {
	cfg := fileConfig{
		Debug:                 &Debug,
		BackupPrivateRepos:    &BackupPrivateRepos,
		ForceRedownload:       &ForceRedownload,
		DeleteDataAfterUpload: &DeleteDataAfterUpload,
		BackupMethods:         []BackupMethod{},
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
