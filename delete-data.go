package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/itsvyle/gh-backup/config"
	log "github.com/sirupsen/logrus"
)

func PostUploadDeleteData() {
	log.Info("Deleting data after upload")

	// List folders in the storage path
	files, err := os.ReadDir(config.LocalStoragePath)
	if err != nil {
		log.WithError(err).Fatal("failed reading directory")
	}

	for _, file := range files {
		if file.IsDir() {
			p := filepath.Join(config.LocalStoragePath, file.Name())
			err = deleteData(p)
			if err != nil {
				log.WithField("directory", p).WithError(err).Fatal("failed removing directory")
			}
		}
	}
}

func deleteData(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed reading directory: %w", err)
	}
	// TODO: make it concurrent
	for _, file := range files {
		if file.IsDir() {
			err = os.RemoveAll(filepath.Join(dir, file.Name()))
			if err != nil {
				return fmt.Errorf("failed removing directory %s: %w", file.Name(), err)
			}
			continue
		}
		if file.Name() == config.BackupInfoFile {
			continue
		}
		path := filepath.Join(dir, file.Name())
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed removing file %s: %w", path, err)
		}
	}
	return nil
}
