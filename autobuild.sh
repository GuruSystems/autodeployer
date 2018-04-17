#!/bin/sh
# go insists on absolute path.

export GOOS=linux
export GOARCH=amd64

[ -z "${BUILD_NUMBER}" ] && export BUILD_NUMBER=${CI_PIPELINE_ID}
[ -z "${PROJECT_NAME}" ] && export PROJECT_NAME=${CI_PROJECT_NAME}
[ -z "${COMMIT_ID}" ] && export COMMIT_ID=${CI_COMMIT_SHA}
[ -z "${GIT_BRANCH}" ] && export GIT_BRANCH=${CI_COMMIT_REF_NAME}


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
export GOBIN=${GOBIN}/darwin
MYSRC=src/golang.conradwood.net/deploymonkey
( cd ${MYSRC} && make client ) || exit 10



build-repo-client -branch=${GIT_BRANCH} -build=${BUILD_NUMBER} -commitid=${COMMIT_ID} -commitmsg="commit msg unknown" -repository=${PROJECT_NAME} -server_addr=buildrepo:5004 

