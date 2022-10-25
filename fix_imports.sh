#!/bin/sh
# Time-stamp: <2022-10-24 23:57:57 krylon>

/usr/bin/env perl -pi'.bak' -e 's{"guang/(\w+)"}{"github.com/blicero/guang/$1"}' $@


