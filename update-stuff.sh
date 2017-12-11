#!/bin/sh
TDIR=src/golang.conradwood.net/vendor/golang.conradwood.net/
mkdir -p ${TDIR}
rsync -pvra --delete --exclude "vendor" --exclude ".git" /home/cnw/devel/picoservices/src/golang.conradwood.net/ ${TDIR}
( cd $TDIR ; find -name '*.go' |xargs -n50 git add )

TDIR=src/golang.conradwood.net/vendor/golang.conradwood.net/
mkdir -p ${TDIR}
rsync -pvra --exclude "vendor" --exclude ".git" /home/cnw/devel/logservice/src/golang.conradwood.net/ ${TDIR}
( cd $TDIR ; find -name '*.go' |xargs -n50 git add )
