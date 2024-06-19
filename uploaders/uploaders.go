package uploaders

import "time"

type Uploader interface {
	Type() string
	// Returns the name of the uploader to be displayed
	Name() string
	// Returns whether the uploader is enabled
	Enabled() bool
	// Checks that the uploader is properly configured and can connect to the remote service; this includes authentication
	Connect() error
	// Basically returns the top level gh_backup_info.json file
	GetPreviousBackupTimes() (map[string]time.Time, error)
	// Push the backup to the remote service
	Push(changedRepos []string, infoFile map[string]time.Time) error
}
