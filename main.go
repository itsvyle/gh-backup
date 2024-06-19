package main

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/cli/go-gh/v2"
	"github.com/itsvyle/gh-backup/config"
	log "github.com/sirupsen/logrus"
)

type Repo struct {
	Name        string `json:"nameWithOwner"`
	NameNoOwner string `json:"name"`
	IsPrivate   bool   `json:"isPrivate"`
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
	reposFields := []string{"name", "nameWithOwner", "isPrivate", "owner"}
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
	for _, repo := range repos {
		currentlyProcessing <- struct{}{}
		go func(repo Repo) {
			defer func() {
				<-currentlyProcessing
				wg.Done()
			}()
			log.Info("Downloading repo: ", repo.Name)
			err := DownloadRepo(repo)
			if err != nil {
				log.WithError(err).Error("failed to download repo")
			}
		}(repo)
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
			log.WithField("repo", repo.Name).WithField("entries", entries).Debug("repo already exists, skipping")
			return nil
		}
	}
	_, _, err := gh.Exec("repo", "clone", repo.Name, path)
	return err
}
