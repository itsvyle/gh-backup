package uploaders

import (
	"github.com/itsvyle/gh-backup/config"
	log "github.com/sirupsen/logrus"
)

type UploaderLocalFolders struct {
	name       string
	enabled    bool
	FolderPath string `yaml:"path"`
}

func NewUploaderLocalFolders(settings *config.BackupMethod) *UploaderLocalFolders {
	if settings.Parameters == nil {
		settings.Parameters = make(map[string]string)
	}
	if settings.Parameters["path"] == "" {
		log.WithField("name", settings.Name).Error("Local folder path not set")
	}

	return &UploaderLocalFolders{
		name:       settings.Name,
		enabled:    settings.Enabled,
		FolderPath: settings.Parameters["path"],
	}
}

func (u *UploaderLocalFolders) Type() string {
	return "local"
}

func (u *UploaderLocalFolders) Name() string {
	return u.name
}

func (u *UploaderLocalFolders) Enabled() bool {
	return u.enabled
}
