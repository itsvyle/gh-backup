# gh-backup
Backup all your github repositories to various places, including cloud storage

The point of this program isn't to just backup the repositories contents, like you can do by simply downloading a zip file of the repository; this also backs up the git history.

## Install
For now, this program has only be tested on Linux.

You can download a binary from the [releases page](https://github.com/itsvyle/gh-backup/releases), and place that binary in your /usr/local/bin directory.

You can also build from source:
```bash
git clone https://github.com/itsvyle/gh-backup.git
cd gh-backup
go build -o ./gh-backup
sudo mv ./gh-backup /usr/local/bin
```

## Requirements
To use this program, you will have to have the Github CLI installed, and be logged in to it. You can install it here: https://cli.github.com/

Once it's installed, you can log in with:
```bash
gh auth login
```

## Configuration

Configuration can either be done in a yaml file at the path `~/.ghbackup.yaml`, or by passing flags to the program.

| Option name | Description | Configuration file key | Flag | Default | 
| --- | --- | --- | --- | --- |
| Backup private repositories | Backup private repositories | backupPrivateRepos | --private | true | 
| Concurrent repositories download | Number of repositories to download from github concurrently | concurrentRepos | --concurrent | 5 |
| Force download | Force download of repositories, even if it hasn't changed since the last download | forceRedownload | --no-cache | false |
| Delete temp files | Delete temporary files created during the backup process after the repos have been uploaded | deleteDataAfterUpload | --delete-temp-files | true |
| Non Interactive | Runs the app without any interaction requirement; this means that it will crash itself if the setup isn't correct; useful for automated runs | nonInteractive | --non-interactive | false |

You can then have a list of backup methods, which can be configured in the configuration file. Each backup method has a name, a type, and parameters specific to that type; see below for supported upload methods.

Example file:
```yaml
backupPrivateRepos: true
backupOtherOwnersRepos: false
concurrentRepoDownloads: 5
forceRedownload: false
backupMethods:
  - name: TEST
    enabled: true
    type: local
    parameters:
      path: /tmp/test
  - name: GDRIVE
    enabled: true
    type: gdrive
    parameters:
      clientID: ********.apps.googleusercontent.com
      clientSecret: ********
```

## Supported upload methods
The program supports multiple upload methods, which can be configured in the configuration file.

Backup methods can be added to the config file in the following format:
```yaml
backupMethods:
  - name: <name>
    enabled: <true/false>
    type: <type>
    parameters:
      <parameter>: <value>
```

### Local directory (`type: local`)
Just copies the repositories to a local directory on the system
| Parameter | Description | Required | Default |
| --- | --- | --- | --- |
| path | Path to the directory to copy the repositories to | true | - |

### Google Drive (`type: gdrive`)
Uploads the repositories to a Google Drive folder

To do this you will need to create a project in the Google Cloud Console, and enable the Google Drive API for that project. You will also need to create OAuth 2.0 credentials for the project, and download the client ID and client secret.
When creating the OAuth 2.0 credentials, you will need to set the application type to "TVs and Limited Input devices", and set the user type to "Internal" in the OAUTH consent screen section

The first time you run the program you will need to authenticate with google drive.

| Parameter | Description | Required | Default |
| --- | --- | --- | --- |
| clientID | Google Drive client ID | true | - |
| clientSecret | Google Drive client secret | true | - |