#!/bin/bash

git diff --quiet HEAD
code=$?

if [ "$code" != "0" ]; then
    echo "Error: Can not build: repository is dirty."
    exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
DATE=$(date +'%Y.%m.%d')
COMMIT_HASH=$(git rev-parse --short HEAD)

# Get interpolator from https://github.com/onuryurdupak/interpolator
# shellcheck disable=SC2016
interpolator "$REPO_ROOT/fake-proxy/embed/data.go" ':=' 'Stamp_build_date\s+=\s+"\${build_date}":=Stamp_build_date = '\""$DATE"\"
code=$?
if [ "$code" != "0" ]; then
    echo "Error: Attempt to run interpolator exited with code: $code."
    exit $code
fi

# shellcheck disable=SC2016
interpolator "$REPO_ROOT/fake-proxy/embed/data.go" ':=' 'Stamp_commit_hash\s+=\s+"\${commit_hash}":=Stamp_commit_hash = '\""$COMMIT_HASH"\"
code=$?
if [ "$code" != "0" ]; then
    echo "Error: Attempt to run interpolator exited with code: $code."
    exit $code
fi

go env -w GOOS=linux GOARCH=amd64
go build

go env -w GOOS=windows GOARCH=amd64
go build

git reset --hard
