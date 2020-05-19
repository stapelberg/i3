#!/bin/sh

set -eu

distri install iputils
tracepath repo.distr1.org

mkdir /lib64 && ln -s /ro/glibc-amd64-2.31-4/out/lib/ld-linux-x86-64.so.2 /lib64/ld-linux-x86-64.so.2 && distri install asciidoc xmlto meson gcc binutils startup-notification xcb-util-cursor xcb-util-keysyms xcb-util-xrm xcb-util-wm libxkbcommon yajl2 pcre pango fribidi harfbuzz libx11 fontconfig libev libxcb perl make && (rm -rf build; mkdir -p build && cd build && (CFLAGS="-Wformat -Wformat-security -Wextra -Wno-unused-parameter -Wstrict-prototypes -Wmissing-prototypes -Werror -fno-common" meson .. -Ddocs=true -Dmans=false -Db_sanitize=address) && ninja -v)
