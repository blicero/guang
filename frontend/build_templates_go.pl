#!/usr/bin/env perl
# -*- mode: cperl; coding: utf-8; -*-
# /Users/krylon/go/src/videostore/build_templates_go.pl
# created at 05. 09. 2015 by Benjamin Walkenhorst
# (c) 2015 Benjamin Walkenhorst <krylon@gmx.net>
# Time-stamp: <2016-02-06 18:32:33 krylon>
#  Redistribution and use in source and binary forms, with or without
#  modification, are permitted provided that the following conditions
#  are met:
#  1. Redistributions of source code must retain the copyright
#     notice, this list of conditions and the following disclaimer.
#  2. Redistributions in binary form must reproduce the above copyright
#     notice, this list of conditions and the following disclaimer in the
#     documentation and/or other materials provided with the distribution.
#
#  THIS SOFTWARE IS PROVIDED BY BENJAMIN WALKENHORST ``AS IS'' AND
#  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
#  IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
#  ARE DISCLAIMED.  IN NO EVENT SHALL THE AUTHOR OR CONTRIBUTORS BE LIABLE
#  FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
#  DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
#  OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
#  HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
#  LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
#  OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
#  SUCH DAMAGE.

use strict;
use warnings;
use diagnostics;
use utf8;
use feature qw(say);
use 5.012;

use Carp;
use English;
use Time::Piece;

my $now = localtime()->strftime('%Y-%m-%d, %H:%M:%S');

my $output_path = "$ENV{GOPATH}/src/guang/frontend/templates.go";
my $html_root = "$ENV{GOPATH}/src/guang/frontend/html";

open(my $output, ">:encoding(UTF-8)", $output_path)
  or croak "Error opening $output_path: $OS_ERROR";

print {$output} "// Generated on $now\n";

print {$output} <<'GO';

package frontend

type HTML struct {
  Static map[string]string
  Templates map[string]string
}

var html_data HTML = HTML{
Static: map[string]string{
GO

opendir(my $static, "$html_root/static")
  or croak "Error opening $html_root/static: $OS_ERROR";

while (readdir $static) {
  next if /^[.]+$/;
  next if /~$/;
  open(my $fh, '<:encoding(UTF-8)', "$html_root/static/$_")
    or croak "Error opening static/$_ - $OS_ERROR";

  local $/;

  my $content = <$fh>;

  close $fh;

  print {$output} <<"GO";
"$_": `$content`,
GO
}

closedir $static;

print {$output} <<'GO';
},

Templates: map[string]string{
GO

opendir(my $templates, "$html_root/templates")
  or croak "Error opening $html_root/templates: $OS_ERROR";

while (readdir $templates) {
  next if /^[.]+$/;
  next if /~$/;
  open(my $fh, '<:encoding(UTF-8)', "$html_root/templates/$_")
    or croak "Error opening $_: $OS_ERROR";

  local $/;
  my $content = <$fh>;

  close $fh;

  print {$output} <<"GO";
"$_": `$content`,
GO
}

print {$output} <<'GO';
},
}
GO

close $output;

system("go fmt");

# Local Variables: #
# compile-command: "perl -c /Users/krylon/go/src/videostore/build_templates_go.pl" #
# End: #
