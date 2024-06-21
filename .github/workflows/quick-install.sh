#!/bin/bash

if ! command -v curl &> /dev/null; then
  echo "Error: curl is not installed. Please install curl and try again."
  exit 1
fi

if ! command -v jq &> /dev/null; then
  echo "Error: jq is not installed. Please install jq and try again."
  exit 1
fi

if [[ $(uname -m) != "x86_64" ]] ; then
    echo "Unsupported architecture"
    exit 1
fi

release_url="https://api.github.com/repos/itsvyle/gh-backup/releases/latest"
release_data=$(curl -sSL $release_url)

echo "Downloading latest release executable..."
echo "Release: $(echo $release_data | jq .name)"
echo "Tag: $(echo $release_data | jq .tag_name)"
echo "Published: $(echo $release_data | jq .published_at)"

echo $release_data > release_data.json
download_url=$(echo $release_data | jq -r '.assets[0].browser_download_url')
if [ -z "$download_url" ]; then
  echo "Error: Failed to find download URL for latest release."
  exit 1
fi
echo "Download URL: $download_url"

curl -SL $download_url -o gh-backup
chmod +x gh-backup
sudo mv gh-backup /usr/local/bin
echo "Moved gh-backup to /usr/local/bin/gh-backup"