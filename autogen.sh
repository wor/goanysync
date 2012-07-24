#!/bin/sh
autoreconf --install
automake --add-missing --copy &>/dev/null
./configure
