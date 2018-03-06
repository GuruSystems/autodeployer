#!/bin/sh

DOM=golang.conradwood.net

/home/cnw/devel/picoservices/copy-to-repo.sh src/${DOM}/vendor/
/home/cnw/devel/logservice/copy-to-repo.sh src/${DOM}/vendor/

echo git adding new files...

find src/${DOM}/vendor -name '*.go' | xargs -n50 git add
find src/${DOM}/vendor -name '*.proto' | xargs -n50 git add

mkdir -p src/${DOM}/vendor/gitlab.gurusys.co.uk/guru/proto/slackgateway
cp -rvf /home/cnw/devel/guru/slack-gateway/vendor/gitlab.gurusys.co.uk/guru/proto/slackgateway/slackgateway.* src/${DOM}/vendor/gitlab.gurusys.co.uk/guru/proto/slackgateway/


echo Done

exit 0
