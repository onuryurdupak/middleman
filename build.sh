#!/bin/bash

DECLARED_VERSION="0.5.0" # Update manually to embed into the binary.
ALLOW_DIRTY_REPO_BUILD="0"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "Error: Can not build: not inside a git repository. Run this script from within the repository."
    exit 1
fi

git diff --quiet HEAD
code=$?

if [ "$ALLOW_DIRTY_REPO_BUILD" = "1" ]; then
    code=0
fi

if [ "$code" != "0" ]; then
    echo "Error: Can not build: repository is dirty."
    exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
DATE=$(date +'%Y.%m.%d')
SOURCE=$(git remote get-url origin)

cd "$REPO_ROOT" || exit 1

# Back up embed.go and restore it however the script ends.
EMBED="$REPO_ROOT/program/embed.go"
cp "$EMBED" "$EMBED.bak" || exit 1
trap 'mv "$EMBED.bak" "$EMBED"' EXIT

# Get interpolator from https://github.com/onuryurdupak/interpolator
interpolator "$EMBED" ':=' 'stamp_build_date\s+=\s+"\${build_date}":=stamp_build_date = '\""$DATE"\"
code=$?
if [ "$code" != "0" ]; then
    echo "Error: Attempt to run interpolator exited with code: $code."
    exit $code
fi

interpolator "$EMBED" ':=' 'stamp_build_version\s+=\s+"\${build_version}":=stamp_build_version = '\""$DECLARED_VERSION"\"
code=$?
if [ "$code" != "0" ]; then
    echo "Error: Attempt to run interpolator exited with code: $code."
    exit $code
fi

interpolator "$EMBED" ':=' 'stamp_source\s+=\s+"\${source}":=stamp_source = '\""$SOURCE"\"
code=$?
if [ "$code" != "0" ]; then
    echo "Error: Attempt to run interpolator exited with code: $code."
    exit $code
fi

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o middleman_win_x64.exe || exit 1
GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -o middleman_linux_x64 || exit 1
GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -o middleman_darwin_arm64 || exit 1
