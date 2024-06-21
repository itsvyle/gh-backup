package uploaders

// Following this: https://developers.google.com/identity/gsi/web/guides/devices
import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/itsvyle/gh-backup/config"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/drive/v3"
)

type UploaderGoogleDrive struct {
	name            string
	enabled         bool
	credentialsFile string
	clientID        string
	clientSecret    string
	driveService    drive.Service
}

func NewUploaderGoogleDrive(settings *config.BackupMethod) *UploaderGoogleDrive {
	if settings.Name == "" {
		log.Fatal("Name not set for Google Drive uploader")
	}
	u := &UploaderGoogleDrive{
		name:    settings.Name,
		enabled: settings.Enabled,
	}

	{
		usr, err := user.Current()
		if err != nil {
			log.WithError(err).Fatal("failed to get current user")
		}

		u.credentialsFile = fmt.Sprintf("%s/.ghbackup/credentials_%s.json", usr.HomeDir, sanitizeName(settings.Name))

		err = os.MkdirAll(fmt.Sprintf("%s/.ghbackup", usr.HomeDir), os.ModePerm)
		if err != nil {
			log.WithError(err).Fatal("failed to create directory")
		}
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
	_, err := os.Stat(u.credentialsFile)
	exists := !errors.Is(err, fs.ErrNotExist)
	if err != nil && exists {
		return fmt.Errorf("credentials file not found: %w", err)
	}

	var idToken GoogleTokenResponse

	if exists {
		log.WithField("name", u.name).Infof("credentials file exists at %s", u.credentialsFile)
		tokenBytes, err := os.ReadFile(u.credentialsFile)
		if err != nil {
			return fmt.Errorf("failed to read token file: %w", err)
		}
		err = json.Unmarshal(tokenBytes, &idToken)
		if err != nil {
			return fmt.Errorf("failed to unmarshal token file: %w", err)
		}
		if idToken.AccessToken == "" {
			return errors.New("access token not found in token file")
		}
	} else {
		idToken, err = u.authenticate()
		if err != nil {
			return err
		}
	}

	if idToken.RefreshedAt.Add(time.Duration(idToken.ExpiresIn) * time.Second).Before(time.Now()) {
		log.WithField("name", u.name).Info("Token expired, refreshing")
		idToken, err = u.refreshToken(idToken)
		if err != nil {
			relogin := config.GetYesNoInput("Failed to refresh token, would you like to re-authenticate?")
			if relogin {
				idToken, err = u.authenticate()
				if err != nil {
					return err
				}
			} else {
				return err
			}
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

type GoogleTokenResponse struct {
	Error        string `json:"error"`
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`

	// Not part of the response
	RefreshedAt time.Time `json:"obtained_at"`
}

func (u *UploaderGoogleDrive) authenticate() (GoogleTokenResponse, error) {
	log.Infof("%s: Authenticating with Google Drive, as the credentials file was not found.", u.name)
	requestBody := fmt.Sprintf("client_id=%s&scope=%s", url.QueryEscape(u.clientID), url.QueryEscape("email https://www.googleapis.com/auth/drive.file"))

	req, err := http.NewRequest(http.MethodPost, "https://oauth2.googleapis.com/device/code", bytes.NewBufferString(requestBody))
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to create request")
		return GoogleTokenResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to send request")
		return GoogleTokenResponse{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to read response body")
		return GoogleTokenResponse{}, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.WithField("name", u.name).WithField("status", resp.StatusCode).WithField("body", string(body)).Error("failed to authenticate")
		return GoogleTokenResponse{}, fmt.Errorf("failed to authenticate: %d", resp.StatusCode)
	}

	var deviceCodeResponse GoogleDriveDeviceCodeResponse
	err = json.Unmarshal(body, &deviceCodeResponse)
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to unmarshal response")
		return GoogleTokenResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	log.Infof("Please visit %s and enter the code: %s", deviceCodeResponse.VerificationURL, deviceCodeResponse.UserCode)

	tokenResponse, err := u.pollTokenAPI(deviceCodeResponse)
	if err != nil {
		return GoogleTokenResponse{}, err
	}
	log.WithField("token", tokenResponse).Info("Authenticated successfully")
	tokenResponse.RefreshedAt = time.Now()

	// Save the contents to the file
	err = u.saveTokenDataToFile(tokenResponse)
	if err != nil {
		return GoogleTokenResponse{}, err
	}

	email, err := u.getEmailFromIDToken(tokenResponse.IDToken)
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to get email from token")
		return GoogleTokenResponse{}, fmt.Errorf("failed to get email from token: %w", err)
	}
	log.Infof("Authenticated as %s", email)

	return tokenResponse, nil
}

const googleDriveTokenTimeToAnswer = 600 // 10 minutes
func (u *UploaderGoogleDrive) pollTokenAPI(deviceCodeResponse GoogleDriveDeviceCodeResponse) (responseToken GoogleTokenResponse, err error) {
	requestBody := fmt.Sprintf(
		"client_id=%s&client_secret=%s&code=%s&grant_type=http://oauth.net/grant_type/device/1.0",
		url.QueryEscape(u.clientID),
		url.QueryEscape(u.clientSecret),
		url.QueryEscape(deviceCodeResponse.DeviceCode),
	)
	req, err := http.NewRequest(http.MethodPost, "https://oauth2.googleapis.com/token", bytes.NewBufferString(requestBody))
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to create request token")
		err = fmt.Errorf("failed to create request: %w", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	maxExpiration := float64(math.Max(float64(deviceCodeResponse.ExpiresIn), googleDriveTokenTimeToAnswer))
	totalChecks := int(math.Floor(maxExpiration / float64(deviceCodeResponse.Interval)))

	client := &http.Client{}
	for i := 0; i < totalChecks; i++ {
		req.Body = io.NopCloser(bytes.NewBufferString(requestBody))
		time.Sleep(time.Duration(deviceCodeResponse.Interval) * time.Second)

		resp, err := client.Do(req)
		if err != nil {
			log.WithField("name", u.name).WithError(err).Error("failed to send request token")
			err = fmt.Errorf("failed to send request: %w", err)
			return responseToken, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusPreconditionRequired {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.WithField("name", u.name).WithField("status", resp.StatusCode).WithField("body", string(body)).Error("failed to poll for token")
			return responseToken, fmt.Errorf("failed to poll for token: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.WithField("name", u.name).WithError(err).Error("failed to read token response body")
			return responseToken, fmt.Errorf("failed to read token response body: %w", err)
		}

		err = json.Unmarshal(body, &responseToken)
		if err != nil {
			log.WithField("name", u.name).WithError(err).Error("failed to unmarshal token response")
			return GoogleTokenResponse{}, fmt.Errorf("failed to unmarshal token response: %w", err)
		}
		return responseToken, nil

	}
	return
}

func (u *UploaderGoogleDrive) refreshToken(tokenData GoogleTokenResponse) (GoogleTokenResponse, error) {
	log.Infof("%s: Obtaining new google drive access token.", u.name)
	requestBody := fmt.Sprintf(
		"client_id=%s&client_secret=%s&refresh_token=%s&grant_type=refresh_token",
		url.QueryEscape(u.clientID),
		url.QueryEscape(u.clientSecret),
		url.QueryEscape(tokenData.RefreshToken),
	)

	req, err := http.NewRequest(http.MethodPost, "https://oauth2.googleapis.com/token", bytes.NewBufferString(requestBody))
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to create request to refresh token")
		return GoogleTokenResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to send request to refresh token")
		return GoogleTokenResponse{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to read response body to refresh token")
		return GoogleTokenResponse{}, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.WithField("name", u.name).WithField("status", resp.StatusCode).WithField("body", string(body)).Error("failed to authenticate")
		return GoogleTokenResponse{}, fmt.Errorf("failed to authenticate: %d", resp.StatusCode)
	}

	var newTokenData GoogleTokenResponse
	err = json.Unmarshal(body, &newTokenData)
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to unmarshal response to refresh token")
		return GoogleTokenResponse{}, errors.New("failed to unmarshal response")
	}

	newTokenData.RefreshToken = tokenData.RefreshToken
	newTokenData.RefreshedAt = time.Now()

	err = u.saveTokenDataToFile(newTokenData)
	if err != nil {
		return GoogleTokenResponse{}, fmt.Errorf("failed to save token data to file: %w", err)
	}
	return newTokenData, nil
}

func (u *UploaderGoogleDrive) saveTokenDataToFile(tokenData GoogleTokenResponse) error {
	tokenBytes, err := json.Marshal(tokenData)
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to marshal token data")
		return fmt.Errorf("failed to marshal token data: %w", err)
	}
	err = os.WriteFile(u.credentialsFile, tokenBytes, 0644)
	if err != nil {
		log.WithField("name", u.name).WithError(err).Error("failed to save token data to file")
		return fmt.Errorf("failed to write token data to file: %w", err)
	}
	return nil
}

func (u *UploaderGoogleDrive) GetPreviousBackupTimes() (res map[string]time.Time, err error) {
	panic("implement me")
}

type GoogleIDToken struct {
	Email string `json:"email"`
}

func (u *UploaderGoogleDrive) getEmailFromIDToken(tokenData string) (email string, err error) {
	parts := strings.Split(tokenData, ".")
	if len(parts) != 3 { //nolint:mnd
		return "", errors.New("invalid token format")
	}
	jwtPayload := parts[1]

	decoded, err := base64.RawURLEncoding.DecodeString(jwtPayload)
	if err != nil {
		return "", fmt.Errorf("failed to decode token payload: %w", err)
	}
	var token GoogleIDToken
	err = json.Unmarshal(decoded, &token)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal token: %w", err)
	}
	if token.Email == "" {
		return "", errors.New("email not found in token")
	}
	return token.Email, nil
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
