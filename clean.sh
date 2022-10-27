#!/bin/sh
# Time-stamp: <2022-10-27 18:42:06 krylon>

cd $GOPATH/src/github.com/blicero/guang/

rm -vf bak.guang guang dbg.build.log \
    && du -sh . \
    && git fsck --full \
    && git reflog expire --expire=now \
    && git gc --aggressive --prune=now \
    && du -sh .


