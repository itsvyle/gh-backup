package uploaders

// Following this: https://developers.google.com/identity/gsi/web/guides/devices
import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/itsvyle/gh-backup/config"
	log "github.com/sirupsen/logrus"
)

type UploaderGoogleDrive struct {
	name         string
	enabled      bool
	clientID     string
	clientSecret string
}

func NewUploaderGoogleDrive(settings *config.BackupMethod) *UploaderGoogleDrive {
	u := &UploaderGoogleDrive{
		name:    settings.Name,
		enabled: settings.Enabled,
	}

	if settings.Parameters == nil {
		settings.Parameters = make(map[string]string)
	}
	if settings.Parameters["clientID"] == "" {
		log.WithField("name", settings.Name).Error("clientID param not set; please add it to the config file after following instructions on https://developers.google.com/identity/gsi/web/guides/devices")
		u.enabled = false
	} else {
		u.clientID = settings.Parameters["clientID"]
	}
	if settings.Parameters["clientSecret"] == "" {
		log.WithField("name", settings.Name).Error("clientSecret param not set; please add it to the config file after following instructions on https://developers.google.com/identity/gsi/web/guides/devices.")
		u.enabled = false
	} else {
		u.clientSecret = settings.Parameters["clientSecret"]
	}

	return u
}

func (u *UploaderGoogleDrive) Type() string {
	return "google-drive"
}

func (u *UploaderGoogleDrive) Name() string {
	return u.name
}

func (u *UploaderGoogleDrive) Enabled() bool {
	return u.enabled
}

func (u *UploaderGoogleDrive) Connect() error {
	credentialsFile := "/etc/gh-backup/" + sanitizeName(u.name) + ".json"
	_, err := os.Stat(credentialsFile)
	exists := !errors.Is(err, fs.ErrNotExist)
	if err != nil && exists {
		return fmt.Errorf("credentials file not found: %w", err)
	}

	if exists {
		log.WithField("name", u.name).Infof("credentials file exists at %s", credentialsFile)
	} else {
		err := u.authenticate()
		if err != nil {
			return err
		}
	}

	return nil
}

type GoogleDriveDeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

func (u *UploaderGoogleDrive) authenticate() error {
	log.Info("Authenticating with Google Drive, as the credentials file was not found.")
	requestBody := fmt.Sprintf("client_id=%s&scope=%s", url.QueryEscape(u.clientID), url.QueryEscape("email https://www.googleapis.com/auth/drive.file"))

	req, err := http.NewRequest(http.MethodPost, "https://oauth2.googleapis.com/device/code", bytes.NewBufferString(requestBody))
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to create request")
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to send request")
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to read response body")
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.WithField("name", u.name).WithField("status", resp.StatusCode).WithField("body", string(body)).Error("failed to authenticate")
		return fmt.Errorf("failed to authenticate: %d", resp.StatusCode)
	}

	var deviceCodeResponse GoogleDriveDeviceCodeResponse
	err = json.Unmarshal(body, &deviceCodeResponse)
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to unmarshal response")
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	log.Infof("Please visit %s and enter the code: %s", deviceCodeResponse.VerificationURL, deviceCodeResponse.UserCode)

	return nil
}

func (u *UploaderGoogleDrive) GetPreviousBackupTimes() (res map[string]time.Time, err error) {
	panic("implement me")
}

func (u *UploaderGoogleDrive) Push(changedRepos []string, infoFile map[string]time.Time) (err error) {

	wg := sync.WaitGroup{}
	wg.Add(len(changedRepos))
	runningCopies := make(chan struct{}, config.ConcurrentRepoDownloads)

	for _, repo := range changedRepos {
		runningCopies <- struct{}{}
		go func(r string) {
			defer func() {
				<-runningCopies
				wg.Done()
			}()
			err = u.pushRepo(r)
			if err != nil {
				log.WithField("name", u.name).WithField("repo", r).WithError(err).Error("failed to copy repo")
			}
		}(repo)
	}

	wg.Wait()

	// Save last updated time for all repos in a file
	// reposUpdatedBytes, err := json.Marshal(infoFile)
	if err != nil {
		return fmt.Errorf("failed to marshal updated repos: %w", err)
	}

	// push the file

	return
}

func (u *UploaderGoogleDrive) pushRepo(repo string) error {
	panic("implement me")
}
