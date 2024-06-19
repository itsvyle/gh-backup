package config

import (
	"flag"
	"strings"
)

const BackupInfoFile = "gh_backup_info.json"

var (
	NonInteractive          = false
	BackupPrivateRepos      = true
	BackupOtherOwnersRepos  = false
	ConcurrentRepoDownloads = 5
	LocalStoragePath        = "/tmp/ghbackup"
	ForceRedownload         = false
	DeleteDataAfterUpload   = true
)

var force = flag.Bool("force", false, "Force redownload of all repos")
var nonInteractive = flag.Bool("nonInteractive", false, "Run in non-interactive mode")

func init() {
	flag.Parse()
	LoadConfig()
	ForceRedownload = *force
	NonInteractive = *nonInteractive

	// Potential processing of arguments:
	LocalStoragePath = strings.TrimSuffix(LocalStoragePath, "/")
}

func SanitizeRepoName(repoName string) string {
	return strings.Replace(repoName, "/", "_", -1)
}
