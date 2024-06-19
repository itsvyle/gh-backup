package config

import (
	"flag"
	"strings"
)

var (
	BackupPrivateRepos      = true
	BackupOtherOwnersRepos  = false
	ConcurrentRepoDownloads = 5
	LocalStoragePath        = "/tmp/ghbackup"
	ForceRedownload         = false
)

var force = flag.Bool("force", false, "Force redownload of all repos")

func init() {
	flag.Parse()
	LoadConfig()
	ForceRedownload = *force

	// Potential processing of arguments:
	LocalStoragePath = strings.TrimSuffix(LocalStoragePath, "/")
}
