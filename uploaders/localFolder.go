package uploaders

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/daemon/graphdriver/copy"
	"github.com/itsvyle/gh-backup/config"
	log "github.com/sirupsen/logrus"
)

type UploaderLocalFolders struct {
	name       string
	enabled    bool
	FolderPath string
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
		log.WithField("type", u.Type()).WithField("name", u.Name()).Debugf("Directory created successfully %s", path)
	} else if err != nil {
		return fmt.Errorf("failed to check path: %w", err)
	}

	// Check write permissions
	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to check path: %w", err)
	}
	if !stat.IsDir() {
		return errors.New("path is not a directory")
	}
	if stat.Mode().Perm()&0200 == 0 {
		return errors.New("no write permissions on directory")
	}
	return nil
}

func (u *UploaderLocalFolders) GetPreviousBackupTimes() (res map[string]time.Time, err error) {
	_, err = os.Stat(u.FolderPath + "/" + config.BackupInfoFile)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]time.Time{}, nil
		}
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

func (u *UploaderLocalFolders) Push(changedRepos []string, infoFile map[string]time.Time) (err error) {

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
				log.WithField("name", u.name).WithField("repo", r).WithError(err).Error("failed to copy repo")
			}
		}(repo)
	}

	wg.Wait()

	// Save last updated time for all repos in a file
	reposUpdatedBytes, err := json.Marshal(infoFile)
	if err != nil {
		return fmt.Errorf("failed to marshal updated repos: %w", err)
	}
	err = os.WriteFile(u.FolderPath+"/"+config.BackupInfoFile, reposUpdatedBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated repos to file: %w", err)
	}

	return
}

func (u *UploaderLocalFolders) pushRepo(repo string) error {
	path := config.LocalStoragePath + "/" + config.SanitizeRepoName(repo)
	_, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("'%s' failed to check repo path: %w", repo, err)
	}
	newPath := u.FolderPath + "/" + config.SanitizeRepoName(repo)

	err = os.RemoveAll(newPath)
	if err != nil {
		return fmt.Errorf("'%s' failed to remove old repo: %w", repo, err)
	}

	err = copy.DirCopy(path, newPath, copy.Content, false)
	if err != nil {
		return fmt.Errorf("'%s' failed to copy repo: %w", repo, err)
	}
	return nil
}
