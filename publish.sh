#!/bin/sh
# a hack because we don't upgrade all servers automatically yet

rsync -pvra --progress /tmp/go/autodeployer-server lb1:/tmp/
ssh lb1 "sudo systemctl stop autodeployer ; sleep 1 ;  sudo cp /tmp/autodeployer-server /usr/local/bin/ ; sudo systemctl start autodeployer"
