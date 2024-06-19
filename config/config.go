package config

import "flag"

var (
	BackupPrivateRepos      = true
	BackupOtherOwnersRepos  = false
	ConcurrentRepoDownloads = 5
	LocalStoragePath        = "/tmp/ghbackup"
)

func init() {
	flag.Parse()
}
