#!/usr/bin/env bash

if [[ "${TRAVIS}" == "true" ]]; then
  git config --local user.name "Jeeves"
  git config --local user.email "jeeves@form3.tech"
fi

export RELEASE_NUMBER=$([[ $TRAVIS_BRANCH = "master" ]] && echo "1.0" || echo "0.0")
export PRE_RELEASE=$([[ $TRAVIS_BRANCH = "master" ]] && echo "false" || echo "true")
export RELEASE_SUFFIX=$([[ $TRAVIS_BRANCH = "master" ]] && echo "" || echo "-$TRAVIS_BRANCH")
export TRAVIS_TAG=$([[ $TRAVIS_TAG = "" ]] && echo v$RELEASE_NUMBER.$TRAVIS_BUILD_NUMBER-$(git log --format=%h -1)$RELEASE_SUFFIX || echo "$TRAVIS_TAG")
export RELEASE_BODY=$(date +'%Y%m%d%H%M%S')

echo "RELEASE_NUMBER = $RELEASE_NUMBER"
echo "PRE_RELEASE = $PRE_RELEASE"
echo "RELEASE_SUFFIX = $RELEASE_SUFFIX"
echo "TRAVIS_TAG = $TRAVIS_TAG"
echo "RELEASE_BODY = $RELEASE_BODY"
