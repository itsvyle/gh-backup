package main

import (
	"strings"
	"time"

	"github.com/itsvyle/gh-backup/config"
	"github.com/itsvyle/gh-backup/gh"
	log "github.com/sirupsen/logrus"
)

type BackupInfo struct {
	Name        string    `json:"name"`
	NameNoOwner string    `json:"nameNoOwner"`
	BackedUpAt  time.Time `json:"backedUpAt"`
	Description string    `json:"description"`
	Archived    bool      `json:"archived"`
	IsPrivate   bool      `json:"isPrivate"`
}
type ReposGeneralBackupInfos map[string]time.Time

type Repo struct {
	Name        string    `json:"nameWithOwner"`
	NameNoOwner string    `json:"name"`
	IsPrivate   bool      `json:"isPrivate"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Archived    bool      `json:"isArchived"`
	Description string    `json:"description"`

	Changed bool
}

var username string

func init() {
	log.SetLevel(log.DebugLevel)
}

func printSeparator() {
	log.Info("----------------------------------------------------------")
	log.Info("")
	log.Info("----------------------------------------------------------")
}

func main() {
	// Get current user info
	userInfo, _, err := gh.Exec("api", "user", "-q", ".login")
	if err != nil {
		log.WithError(err).Fatal("failed to get user info")
	}
	username = strings.TrimSpace(userInfo.String())
	log.WithField("username", username).Debug("got username")

	repos, info := DownloadRepos()

	printSeparator()

	UploadRepos(&repos, info)

	if config.DeleteDataAfterUpload {
		printSeparator()
		PostUploadDeleteData()
	}
}
