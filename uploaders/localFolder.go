package uploaders

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/itsvyle/gh-backup/config"
	log "github.com/sirupsen/logrus"
)

type UploaderLocalFolders struct {
	name       string
	enabled    bool
	FolderPath string `yaml:"path"`
}

func NewUploaderLocalFolders(settings *config.BackupMethod) *UploaderLocalFolders {
	u := &UploaderLocalFolders{
		name:       settings.Name,
		enabled:    settings.Enabled,
		FolderPath: "",
	}
	if settings.Parameters == nil {
		settings.Parameters = make(map[string]string)
	}
	if settings.Parameters["path"] == "" {
		log.WithField("name", settings.Name).Error("Local folder path not set")
		u.enabled = false
	} else {
		u.FolderPath = strings.TrimSuffix(settings.Parameters["path"], "/")
	}

	return u
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

func (u *UploaderLocalFolders) Connect() error {
	path := u.FolderPath
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Path does not exist, create it with write permissions
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
		fmt.Println("Directory created successfully:", path)
	} else if err != nil {
		return fmt.Errorf("failed to check path: %w", err)
	}

	// Check write permissions
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to check path: %w", err)
	}
	if !stat.IsDir() {
		return fmt.Errorf("path is not a directory")
	}
	if stat.Mode().Perm()&0200 == 0 {
		return fmt.Errorf("no write permissions on directory")
	}
	return nil
}

func (u *UploaderLocalFolders) GetPreviousBackupTimes() (res map[string]time.Time, err error) {
	_, err = os.Stat(u.FolderPath + "/" + config.BackupInfoFile)
	if err != nil {
		return nil, err
	}

	file, err := os.ReadFile(u.FolderPath + "/" + config.BackupInfoFile)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(file, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (u *UploaderLocalFolders) Push(changedRepos []string) (err error) {

	wg := sync.WaitGroup{}
	wg.Add(len(changedRepos))
	runningCopies := make(chan struct{}, config.ConcurrentRepoDownloads)

	for _, repo := range changedRepos {
		runningCopies <- struct{}{}
		go func(r string) {
			defer func() {
				<-runningCopies
				wg.Done()
			}()
			err = u.pushRepo(r)
			if err != nil {

			}
		}(repo)
	}
	return nil
}

func (u *UploaderLocalFolders) pushRepo(repo string) error {
	path := config.LocalStoragePath + "/" + config.SanitizeRepoName(repo)
	_, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("'%s' failed to check repo path: %w", repo, err)
	}

	err = os.Rename(path, u.FolderPath+"/"+config.SanitizeRepoName(repo))
	if err != nil {
		return fmt.Errorf("'%s' failed to copy repo: %w", repo, err)
	}
	return nil
}
