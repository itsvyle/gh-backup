package config

import (
	"flag"
	"fmt"
	"strings"
)

const BackupInfoFile = "gh_backup_info.json"

var (
	NonInteractive            = false
	BackupPrivateRepos        = true
	BackupOtherOwnersRepos    = false
	ConcurrentRepoDownloads   = 5
	ConcurrentBackupUploaders = 2
	LocalStoragePath          = "/tmp/ghbackup"
	ForceRedownload           = false
	DeleteDataAfterUpload     = true
	BackupMethods             []BackupMethod
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

func GetYesNoInput(question string) bool {
	if NonInteractive {
		return true
	}
	var response string
	for {
		fmt.Print(question + " (y/n): ")
		_, err := fmt.Scanf("%s", &response)
		if err != nil && err.Error() != "unexpected newline" {
			continue
		}
		if response == "n" || response == "N" {
			return false
		}
		if response == "y" || response == "Y" || response == "" {
			return true
		}
	}
}
