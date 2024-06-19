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

const backupInfoFile = "gh_backup_info.json"

type BackupInfo struct {
	Name        string    `json:"name"`
	NameNoOwner string    `json:"nameNoOwner"`
	BackedUpAt  time.Time `json:"backedUpAt"`
	Description string    `json:"description"`
	Archived    bool      `json:"archived"`
	IsPrivate   bool      `json:"isPrivate"`
}

type Repo struct {
	Name        string    `json:"nameWithOwner"`
	NameNoOwner string    `json:"name"`
	IsPrivate   bool      `json:"isPrivate"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Archived    bool      `json:"isArchived"`
	Description string    `json:"description"`
}

var username string

func init() {
	log.SetLevel(log.DebugLevel)
}

func main() {
	// Get current user info
	userInfo, _, err := gh.Exec("api", "user", "-q", ".login")
	if err != nil {
		log.WithError(err).Fatal("failed to get user info")
	}
	username = strings.TrimSpace(userInfo.String())
	log.WithField("username", username).Debug("got username")

	DownloadRepos()

}

func DownloadRepos() {
	// List repos
	reposFields := []string{"name", "nameWithOwner", "isPrivate", "owner", "updatedAt", "isArchived", "description"}
	reposList, _, err := gh.Exec("repo", "list", "--limit", "500", "--json", strings.Join(reposFields, ","))
	if err != nil {
		log.WithError(err).Fatal("failed to list repos")
	}
	var repos []Repo
	err = json.Unmarshal(reposList.Bytes(), &repos)
	if err != nil {
		log.WithError(err).Error("failed to unmarshal repos")
		return
	}
	log.Infof("Found %d repos", len(repos))
	repos = FilterRepos(repos)
	log.Infof("Filtered down to %d repos", len(repos))

	log.Debugf("Download repos %d at a time to %s", config.ConcurrentRepoDownloads, config.LocalStoragePath)

	reposUpdated := make(map[string]time.Time)

	// Download all repos
	var wg sync.WaitGroup
	wg.Add(len(repos))
	currentlyProcessing := make(chan struct{}, config.ConcurrentRepoDownloads)
	i := 0
	for _, repo := range repos {
		currentlyProcessing <- struct{}{}
		i++
		go func(repo Repo) {
			defer func() {
				<-currentlyProcessing
				wg.Done()
			}()
			err := DownloadRepo(&repo)
			if err != nil {
				log.WithField("repo", repo.Name).WithError(err).Error("failed to download repo")
			}
			reposUpdated[sanitizeRepoName(repo.Name)] = time.Now()
		}(repo)
	}
	wg.Wait()

	log.Info("Finished downloading all repos")

	// Save last updated time for all repos in a file
	reposUpdatedBytes, err := json.Marshal(reposUpdated)
	if err != nil {
		log.WithError(err).Fatal("failed to marshal updated repos")
		return
	}
	err = os.WriteFile(config.LocalStoragePath+"/"+backupInfoFile, reposUpdatedBytes, 0644)
	if err != nil {
		log.WithError(err).Fatal("failed to write updated repos to file")
	}
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
		repos = append(repos, repo)
	}
	return repos
}

func sanitizeRepoName(repoName string) string {
	return strings.Replace(repoName, "/", "_", -1)
}

func DownloadRepo(repo *Repo) error {
	path := config.LocalStoragePath + "/" + sanitizeRepoName(repo.Name)
	folderExists := getFolderExists(path)
	if !config.ForceRedownload && folderExists {
		// check if folder exists already
		entries, err := os.ReadDir(path)
		if err == nil && len(entries) > 0 {
			for _, entry := range entries {
				if entry.Name() == backupInfoFile {
					file, err := os.ReadFile(path + "/" + backupInfoFile)
					if err != nil {
						log.WithField("repo", repo.Name).WithError(err).Error("failed to read backup info file")
						return err
					}
					var backupInfo BackupInfo
					err = json.Unmarshal(file, &backupInfo)
					if err != nil {
						log.WithField("repo", repo.Name).WithError(err).Error("failed to unmarshal backup info")
						return err
					}
					if backupInfo.BackedUpAt.After(repo.UpdatedAt) {
						log.Infof("Repo %s hasn't been modified", repo.Name)
						return nil
					}
				}
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
	err = os.WriteFile(path+"/"+backupInfoFile, backupInfoBytes, 0644)
	if err != nil {
		return err
	}

	return err
}

func getFolderExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
