package main

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cli/go-gh/v2"
	"github.com/itsvyle/gh-backup/config"
	log "github.com/sirupsen/logrus"
)

const backupInfoFile = "gh_backup_info.json"

type BackupInfo struct {
	BackedUpAt time.Time `json:"backedUpAt"`
}

type Repo struct {
	Name        string    `json:"nameWithOwner"`
	NameNoOwner string    `json:"name"`
	IsPrivate   bool      `json:"isPrivate"`
	UpdatedAt   time.Time `json:"updatedAt"`
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

	// List repos
	reposFields := []string{"name", "nameWithOwner", "isPrivate", "owner", "updatedAt"}
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

	// Download all repos
	var wg sync.WaitGroup
	wg.Add(len(repos))
	currentlyProcessing := make(chan struct{}, config.ConcurrentRepoDownloads)
	i := 0
	for _, repo := range repos {
		currentlyProcessing <- struct{}{}
		i++
		go func(repo Repo, i int) {
			defer func() {
				<-currentlyProcessing
				wg.Done()
			}()
			log.Infof("(%d/%d) Downloading repo %s", i, len(repos), repo.Name)
			err := DownloadRepo(repo)
			if err != nil {
				log.WithField("repo", repo.Name).WithError(err).Error("failed to download repo")
			}
		}(repo, i)
	}
	wg.Wait()
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

func DownloadRepo(repo Repo) error {
	path := config.LocalStoragePath + "/" + sanitizeRepoName(repo.Name)
	if !config.ForceRedownload {
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
						log.WithField("repo", repo.Name).Debug("repo is up to date, skipping")
						return nil
					}
				}
			}
		}
	}
	_, _, err := gh.Exec("repo", "clone", repo.Name, path)
	if err != nil {
		return err
	}
	backupInfo := BackupInfo{
		BackedUpAt: time.Now(),
	}
	backupInfoBytes, err := json.Marshal(backupInfo)
	if err != nil {
		return err
	}
	err = os.WriteFile(path+"/"+backupInfoFile, backupInfoBytes, 0644) //nolint:mnd
	if err != nil {
		return err
	}

	return err
}
