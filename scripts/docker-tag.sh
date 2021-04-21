#!/bin/bash

if [ -z ${TRAVIS_PULL_REQUEST+x} ]; then
	TRAVIS_PULL_REQUEST="false"
fi

if [ -z ${TRAVIS_PULL_REQUEST_BRANCH+x} ]; then
	TRAVIS_PULL_REQUEST_BRANCH="develop"
fi

if [ "$TRAVIS_PULL_REQUEST" = "false" ]; then
	TAG=$TRAVIS_BRANCH
else
	TAG="$TRAVIS_PULL_REQUEST_BRANCH-pr"
fi

if [ "$TAG" = "" ]; then
    TAG=$(git rev-parse --abbrev-ref HEAD)
fi

if [ "$TAG" = "master" ]; then
    TAG="latest"
fi
