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
)

func init() {
	flag.Parse()
}

func InitConfig() {
	LocalStoragePath = strings.TrimSuffix(LocalStoragePath, "/")
}
