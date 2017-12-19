#!/bin/sh
# this is a hack because the build server for this repo doesn't build 64 bit properly

rsync -pvra --progress /tmp/go/autodeployer-server master.fra:/tmp/
ssh master.fra "sudo cp /tmp/autodeployer-server /srv/stdfiles/usr/local/bin/"
