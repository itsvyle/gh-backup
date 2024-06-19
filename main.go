package main

import (
	"encoding/json"
	"strings"

	"github.com/cli/go-gh/v2"
	log "github.com/sirupsen/logrus"
)

type Repo struct {
	Name        string `json:"nameWithOwner"`
	NameNoOwner string `json:"name"`
	IsPrivate   bool   `json:"isPrivate"`
}

func main() {
	reposFields := []string{"name", "nameWithOwner", "isPrivate"}
	reposList, _, err := gh.Exec("repo", "list", "--json", strings.Join(reposFields, ","))
	if err != nil {
		log.WithError(err).Fatal("failed to list repos")
	}
	var repos []Repo
	err = json.Unmarshal(reposList.Bytes(), &repos)
	if err != nil {
		log.WithError(err).Error("failed to unmarshal repos")
		return
	}
	log.WithField("repos", repos).Info("got repos")
}
