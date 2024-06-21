package uploaders

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"

	"github.com/itsvyle/gh-backup/config"
)

func GetZipPath(repoName string) (string, error) {
	zipPath := config.LocalStoragePath + "/zips/" + config.SanitizeRepoName(repoName) + ".zip"
	_, err := os.Stat(zipPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("'%s' failed to check zip path: %w", repoName, err)
		}
		sourcePath := config.LocalStoragePath + "/" + config.SanitizeRepoName(repoName)
		_, err := os.Stat(sourcePath)
		if err != nil {
			return "", fmt.Errorf("'%s' failed to check repo source path path: %w", repoName, err)
		}
		err = ZipFolder(sourcePath, zipPath)
		if err != nil {
			return "", fmt.Errorf("'%s' failed to zip folder: %w", repoName, err)
		}
	}
	return zipPath, nil
}

func ZipFolder(sourcePath, zipPath string) error {
	cmd := exec.Command("zip", "-r", zipPath, sourcePath)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run zip command: %w", err)
	}
	return nil
}
