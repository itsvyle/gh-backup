package config

import (
	"flag"
	"strings"
)

var (
	BackupPrivateRepos        = true
	BackupOtherOwnersRepos    = false
	ConcurrentRepoDownloads   = 5
	LocalStoragePath          = "/tmp/ghbackup"
	ForceRedownload           = false
	BranchesToCheckForChanges = []string{"master", "main"}
)

func init() {
	flag.Parse()
}

func InitConfig() {
	LocalStoragePath = strings.TrimSuffix(LocalStoragePath, "/")
}
