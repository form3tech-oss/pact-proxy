#!/bin/bash

set -e

CURDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
# shellcheck source=./docker-tag.sh
source "$CURDIR/docker-tag.sh"

APPNAME=$(basename $1)
REPOSITORYNAME=$(basename $(cd "$CURDIR/.." && pwd))
VERSION=$TAG

if [ -f "build/package/$APPNAME/Dockerfile" ]; then
    docker build -t tech.form3/$APPNAME:$VERSION --build-arg APPNAME=$APPNAME --build-arg REPOSITORYNAME=$REPOSITORYNAME --build-arg TAGS={version:\"$VERSION\"} -f build/package/$APPNAME/Dockerfile .
    docker tag tech.form3/$APPNAME:$VERSION tech.form3/$APPNAME:$TAG
else
    echo "No Dockerfile found for \"$APPNAME\", not building a Docker image"
fi