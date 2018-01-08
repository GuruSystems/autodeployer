#!/bin/sh

DOM=golang.conradwood.net
/home/cnw/devel/picoservices/copy-to-repo.sh src/${DOM}/vendor/

find src/${DOM}/vendor -name '*.go' | xargs -n50 git add
find src/${DOM}/vendor -name '*.proto' | xargs -n50 git add
exit 0
