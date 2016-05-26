#!/usr/bin/env bash
#
# Shell script for building binaries for all relevant platforms
set -euo pipefail

SCRIPT_DIR=$(dirname "$0")
cd "${SCRIPT_DIR}"
DIR_NAME=${PWD##*/} # name of current directory = name of project

go test -v
if [ "$?" -ne "0" ] ; then
    echo "go test failed, aborting"
    exit 1
fi

export CGO_ENABLED=0

# Build
declare -a TARGETS=(darwin linux freebsd)
export GOARCH=amd64
VERSION=$(git describe --abbrev=0 --tags 2> /dev/null) # this doesn't actually seem to work
# Create a tar-ball for release
if [ "$?" -ne 0 ] ; then
  # No tag, use commit hash
  HASH=$(git rev-parse HEAD)
  VERSION=${HASH:0:7}
fi

for target in "${TARGETS[@]}" ; do
  output="${DIR_NAME}"
  echo "Building for ${target}, output bin/${output}"
  export GOOS=${target}
  export GOARCH=amd64
  go build -o "bin/${output}"
  (
  cd ..
  TARBALL="${DIR_NAME}-${VERSION}-${target}-${GOARCH}.tar.gz"
  tar -cf "${TARBALL}" --exclude=.git -z "${DIR_NAME}"
  echo "Created: ${PWD}/${TARBALL}"
  )
  rm -rf "bin/${output}"
done
