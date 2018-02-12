#!/bin/sh
# go insists on absolute path.

export GOOS=linux
export GOARCH=amd64



export GOBIN=`pwd`/dist

rm -rf dist
mkdir dist

build-repo-client -branch=${GIT_BRANCH} -build=${BUILD_NUMBER} -commitid=${COMMIT_ID} -commitmsg="commit msg unknown" -repository=${PROJECT_NAME} -versiondir=src/ || exit 10


export GOPATH=`pwd`
echo "GOPATH=$GOPATH"
mkdir $GOBIN
MYSRC=src/golang.conradwood.net/deploymonkey
( cd ${MYSRC} && make proto ) || exit 10

MYSRC=src/golang.conradwood.net/autodeployer
( cd ${MYSRC} && make proto ) || exit 10
( cd ${MYSRC} && make client ) || exit 10
( cd ${MYSRC} && make server ) || exit 10
cp -rvf ${MYSRC}/proto dist/

MYSRC=src/golang.conradwood.net/deploymonkey
( cd ${MYSRC} && make client ) || exit 10
( cd ${MYSRC} && make server ) || exit 10
cp -rvf ${MYSRC}/proto dist/


export GOOS=darwin
export GOARCH=amd64
export GOBIN=dist/darwin
MYSRC=src/golang.conradwood.net/deploymonkey
( cd ${MYSRC} && make client ) || exit 10



build-repo-client -branch=${GIT_BRANCH} -build=${BUILD_NUMBER} -commitid=${COMMIT_ID} -commitmsg="commit msg unknown" -repository=${PROJECT_NAME} -server_addr=buildrepo:5004 

