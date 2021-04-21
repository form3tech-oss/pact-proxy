#!/bin/bash

EXTRA_TAGS=,team=euro
docker build \
  -t tech.form3/$(basename $1):$TAG \
  --build-arg REPONAME=sepadd-gateway \
  --build-arg APPNAME=$(basename $1) \
  --build-arg TAGS={version:\"$TAG\"}${EXTRA_TAGS} \
  -f build/package/$(basename $1)/Dockerfile .
