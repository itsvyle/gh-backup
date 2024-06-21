package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/itsvyle/gh-backup/config"
	"github.com/itsvyle/gh-backup/gh"
	log "github.com/sirupsen/logrus"
)

func DownloadRepos() ([]Repo, ReposGeneralBackupInfos) {
	// List repos
	reposFields := []string{"name", "nameWithOwner", "isPrivate", "owner", "updatedAt", "isArchived", "description"}
	reposList, _, err := gh.Exec("repo", "list", "--limit", "500", "--json", strings.Join(reposFields, ","))
	if err != nil {
		log.WithError(err).Fatal("failed to list repos")
	}
	var repos []Repo
	err = json.Unmarshal(reposList.Bytes(), &repos)
	if err != nil {
		log.WithError(err).Fatal("failed to unmarshal repos")
	}
	log.Infof("Found %d repos", len(repos))
	repos = FilterRepos(repos)
	log.Infof("Filtered down to %d repos", len(repos))

	log.Debugf("Download repos %d at a time to %s", config.ConcurrentRepoDownloads, config.LocalStoragePath)

	reposUpdated := make(ReposGeneralBackupInfos)
	// Find the general backup info file
	previousBackupInfoBytes, err := os.ReadFile(config.LocalStoragePath + "/" + config.BackupInfoFile)
	if err == nil {
		err = json.Unmarshal(previousBackupInfoBytes, &reposUpdated)
		if err != nil {
			log.WithError(err).Fatal("failed to unmarshal previous backup info")
		}
	}

	// Download all repos
	var wg sync.WaitGroup
	wg.Add(len(repos))
	currentlyProcessing := make(chan struct{}, config.ConcurrentRepoDownloads)
	for _, repo := range repos {
		currentlyProcessing <- struct{}{}
		go func(repo Repo) {
			defer func() {
				<-currentlyProcessing
				wg.Done()
			}()
			err := DownloadRepo(&repo, &reposUpdated)
			if err != nil {
				log.WithField("repo", repo.Name).WithError(err).Error("failed to download repo")
				return
			}
		}(repo)
	}
	wg.Wait()

	for _, repo := range repos {
		reposUpdated[config.SanitizeRepoName(repo.Name)] = time.Now()
	}

	log.Info("Finished downloading all repos")

	// Save last updated time for all repos in a file
	reposUpdatedBytes, err := json.Marshal(reposUpdated)
	if err != nil {
		log.WithError(err).Fatal("failed to marshal updated repos")
	}
	err = os.WriteFile(config.LocalStoragePath+"/"+config.BackupInfoFile, reposUpdatedBytes, 0644)
	if err != nil {
		log.WithError(err).Fatal("failed to write updated repos to file")
	}

	return repos, reposUpdated
}

func FilterRepos(initialRepos []Repo) []Repo {
	var repos []Repo
	for _, repo := range initialRepos {
		if repo.IsPrivate && !config.BackupPrivateRepos {
			continue
		}
		if !config.BackupOtherOwnersRepos && repo.Name != username+"/"+repo.NameNoOwner {
			continue
		}
		if config.ExludeRepos != nil {
			exclude := false
			for _, excludeRepo := range config.ExludeRepos {
				if excludeRepo == repo.Name {
					exclude = true
					break
				}
			}
			if exclude {
				continue
			}
		}
		repos = append(repos, repo)
	}
	return repos
}

func DownloadRepo(repo *Repo, uploadedTimes *ReposGeneralBackupInfos) error {
	path := config.LocalStoragePath + "/" + config.SanitizeRepoName(repo.Name)
	folderExists := getFolderExists(path)
	if !config.ForceRedownload && folderExists {
		if t, ok := (*uploadedTimes)[config.SanitizeRepoName(repo.Name)]; ok {
			if t.After(repo.UpdatedAt) {
				log.Infof("Repo %s is up to date", repo.Name)
				return nil
			}
		}
	}

	var err error
	var stder bytes.Buffer

	if folderExists {
		err = os.RemoveAll(path)
		if err != nil {
			return err
		}
	}
	log.Infof("Downloading repo %s", repo.Name)
	_, stder, err = gh.Exec("repo", "clone", repo.Name, path)

	if err != nil {
		return errors.New(stder.String())
	}
	repo.Changed = true

	backupInfo := BackupInfo{
		Name:        repo.Name,
		NameNoOwner: repo.NameNoOwner,
		BackedUpAt:  time.Now(),
		Description: repo.Description,
		Archived:    repo.Archived,
		IsPrivate:   repo.IsPrivate,
	}
	backupInfoBytes, err := json.Marshal(backupInfo)
	if err != nil {
		return err
	}
	err = os.WriteFile(path+"/"+config.BackupInfoFile, backupInfoBytes, 0644)
	if err != nil {
		return err
	}

	return err
}

func getFolderExists(path string) bool {
	s, err := os.Stat(path)
	return err == nil && s.IsDir()
}
