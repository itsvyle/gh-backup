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
	Debug                     = false
)

var flagForce = flag.Bool("no-cache", ForceRedownload, "Force redownload of all repos")
var flagNonInteractive = flag.Bool("non-interactive", NonInteractive, "Run in non-interactive mode")
var flagPrivate = flag.Bool("private", BackupPrivateRepos, "Backup private repos")
var flagConcurrent = flag.Int("concurrent", ConcurrentRepoDownloads, "Number of concurrent downloads")
var flagDeleteTempFiles = flag.Bool("delete-temp-files", DeleteDataAfterUpload, "Delete backup temporary files after upload")
var flagDebug = flag.Bool("debug", Debug, "Enable debug mode")

func init() {
	flag.Parse()
	LoadConfig()
	// ForceRedownload = *flagForce
	// NonInteractive = *flagNonInteractive
	// BackupPrivateRepos = *flagPrivate
	// ConcurrentRepoDownloads = *flagConcurrent
	// DeleteDataAfterUpload = *flagDeleteTempFiles
	// Debug = *flagDebug

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
