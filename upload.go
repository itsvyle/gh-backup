package main

import (
	"errors"
	"io/fs"
	"os"
	"sync"

	"github.com/itsvyle/gh-backup/config"
	"github.com/itsvyle/gh-backup/uploaders"
	log "github.com/sirupsen/logrus"
)

func UploadRepos(repos *[]Repo, backupInfoFile ReposGeneralBackupInfos) {
	log.Infof("Found %d upload backup methods ", len(config.BackupMethods))
	if len(config.BackupMethods) == 0 {
		log.Fatal("No backup methods configured. Please add them to ~/" + config.ConfigFileName)
	}

	backs := make([]uploaders.Uploader, len(config.BackupMethods))

	for i, method := range config.BackupMethods {
		switch method.Type {
		case "local":
			backs[i] = uploaders.NewUploaderLocalFolders(&method)
		case "gdrive":
			backs[i] = uploaders.NewUploaderGoogleDrive(&method)
		default:
			log.WithField("type", method.Type).Fatal("Unknown backup method")
		}
	}

	for _, method := range backs {
		log.Printf("- %s: '%s'; enabled=%t", method.Type(), method.Name(), method.Enabled())
	}
	enabledMethodsCount := 0
	for _, method := range backs {
		if method.Enabled() {
			enabledMethodsCount++
			err := method.Connect()
			if err != nil {
				log.WithField("type", method.Type()).WithField("name", method.Name()).WithError(err).Fatal("failed to connect")
			}
		}
	}

	// Create zips folder
	zipsPath := config.LocalStoragePath + "/zips/"
	os.RemoveAll(zipsPath)
	_, err := os.Stat(zipsPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			log.WithError(err).Fatal("failed to check zips path")
		}
		err = os.MkdirAll(zipsPath, os.ModePerm)
		if err != nil {
			log.WithError(err).Fatal("failed to create zips path")
		}
	}

	var hadError error

	wg := sync.WaitGroup{}
	wg.Add(enabledMethodsCount)
	runningUploaders := make(chan struct{}, config.ConcurrentRepoDownloads)

	for _, method := range backs {
		if !method.Enabled() {
			continue
		}
		runningUploaders <- struct{}{}
		go func(method uploaders.Uploader) {
			defer func() {
				<-runningUploaders
				wg.Done()
			}()

			previousBackupTimes, err := method.GetPreviousBackupTimes()
			if err != nil {
				log.WithField("type", method.Type()).WithField("name", method.Name()).WithError(err).Error("failed to get previous backup times")
				hadError = err
				return
			}

			changedReposNames := []string{}
			for _, repo := range *repos {
				if repo.UpdatedAt.After(previousBackupTimes[config.SanitizeRepoName(repo.Name)]) {
					changedReposNames = append(changedReposNames, repo.Name)
				}
			}

			err = method.Push(changedReposNames, backupInfoFile)
			if err != nil {
				log.WithField("type", method.Type()).WithField("name", method.Name()).WithError(err).Error("failed to upload")
				hadError = err
				return
			}
			log.WithField("type", method.Type()).WithField("name", method.Name()).Infof("uploaded %d repos", len(changedReposNames))
		}(method)
	}

	wg.Wait()

	if hadError != nil {
		log.Fatal("Failed to upload some repos")
	}
}
