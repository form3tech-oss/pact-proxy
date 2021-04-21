#!/usr/bin/env bash
#
# gh-dl-release! It works!
#
# This script downloads an asset from latest or specific Github release of a
# private repo. Feel free to extract more of the variables into command line
# parameters.
#
# PREREQUISITES
#
# curl, wget, jq
#
# USAGE
#
# Set all the variables inside the script, make sure you chmod +x it, then
# to download specific version to my_app.tar.gz:
#
#     gh-dl-release 2.1.1 my_app.tar.gz
#
# to download latest version:
#
#     gh-dl-release latest latest.tar.gz
#
# If your version/tag doesn't match, the script will exit with error.

TOKEN=$1                         # git access token
ORG=$2                           # github organisation e.g. "form3tech"
REPO=$3                          # repository to fetch from e.g. "paymentapi"
SOURCE_FILE=$4                   # the name of your release asset file, e.g. build.tar.gz
VERSION=$5                       # tag name or the word "latest"
DEST_FILE=$6                     # destination file
GITHUB="https://api.github.com"

function gh_curl() {
  curl -H "Authorization: token $TOKEN" \
       -H "Accept: application/vnd.github.v3.raw" \
       "$@"
}

if [ "$VERSION" != "latest" ]; then
  TAGS="tags/";
fi;

URL="$GITHUB/repos/$ORG/$REPO/releases/$TAGS$VERSION"


if [ -z "$TOKEN" ]; then
  echo "gh-download-release.sh can't get $URL as GITHUB_TOKEN is not set."
  exit 1
fi;

asset_id=$(gh_curl -s "$URL" | \
  jq ".assets | map(select(.name == \"$SOURCE_FILE\"))[0].id" )

if [ -z "$asset_id" ]; then
  >&2 echo "ERROR: version not found $VERSION"
  exit 1
fi;

wget -q --auth-no-challenge --header='Accept:application/octet-stream' \
  "https://$TOKEN:@api.github.com/repos/$ORG/$REPO/releases/assets/$asset_id" \
  -O "$DEST_FILE"