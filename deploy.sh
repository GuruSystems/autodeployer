#!/bin/sh

# dodgy script, but see readme on target

TARGET=master.fra
ssh ${TARGET} "sudo systemctl stop deploymonkey"
rsync --inplace -pvra dist/deploymonkey-server ${TARGET}:/srv/deployments
ssh ${TARGET} "sudo systemctl start deploymonkey"
