#!/bin/sh
# Time-stamp: <2022-10-27 18:39:44 krylon>

cd $GOPATH/src/github.com/blicero/ticker/

rm -vf bak.guang guang dbg.build.log \
    && du -sh . \
    && git fsck --full \
    && git reflog expire --expire=now \
    && git gc --aggressive --prune=now \
    && du -sh .


