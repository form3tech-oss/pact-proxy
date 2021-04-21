#!/usr/bin/env bash

NAME="$TAG"
BODY="$RELEASE_BODY"
REPO="form3tech/sepadd-gateway"
TERRAFORM_ASSET="$(git rev-parse --show-toplevel)/terraform.tar.gz"
SWAGGER_ASSET="$(git rev-parse --show-toplevel)/swagger.yaml"

# 1. create a draft release
payload=$(
  jq --null-input \
     --arg tag "$TAG" \
     --arg commit "$COMMIT" \
     --arg body "$BODY" \
     --argjson prerelease "$PRE_RELEASE" \
     '{ tag_name: $tag, target_commitish: $commit, body: $body, draft: true, prerelease: $prerelease }'
)

response=$(
  curl -d "$payload" \
       "https://api.github.com/repos/$REPO/releases?access_token=$GITHUB_TOKEN"
)

release_url="$(echo "$response" | jq -r .url)"
upload_url="$(echo "$response" | jq -r .upload_url | sed -e "s/{?name,label}//")"

# 2. upload release assets
curl -H "Content-Type:application/gzip" \
     --data-binary "@$TERRAFORM_ASSET" \
     "$upload_url?name=$(basename "$TERRAFORM_ASSET")&access_token=$GITHUB_TOKEN"

curl -H "Content-Type:application/yaml" \
     --data-binary "@$SWAGGER_ASSET" \
     "$upload_url?name=$(basename "$SWAGGER_ASSET")&access_token=$GITHUB_TOKEN"

# 3. publish the release to trigger a new webhook for app-version-manager that includes the uploaded
# assets
curl -X PATCH \
     -H "Content-Type:application/json" \
     -d '{ "draft": false }' \
     "$release_url?access_token=$GITHUB_TOKEN"
