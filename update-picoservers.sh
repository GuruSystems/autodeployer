#!/bin/sh
TDIR=src/golang.conradwood.net/vendor/golang.conradwood.net/
mkdir -p ${TDIR}
rsync -pvra --delete /home/cnw/devel/picoservices/src/golang.conradwood.net/ ${TDIR}
( cd $TDIR ; find -name '*.go' |xargs -n1 git add )

