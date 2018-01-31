#!/bin/sh

DOM=golang.conradwood.net
M=`pwd`
/home/cnw/devel/picoservices/copy-to-repo.sh ${M}/src/${DOM}/vendor/
/home/cnw/devel/logservice/copy-to-repo.sh ${M}/src/${DOM}/vendor/

echo git adding new files...

find src/${DOM}/vendor -name '*.go' | xargs -n50 git add
find src/${DOM}/vendor -name '*.proto' | xargs -n50 git add


echo Done

exit 0
