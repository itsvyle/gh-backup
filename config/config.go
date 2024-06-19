package config

import "flag"

var (
	BackupPrivateRepos      = true
	ConcurrentRepoDownloads = 5
)

func init() {
	flag.Parse()
}
