#!/bin/sh
autoreconf --install
automake --add-missing --copy 2>/dev/null
./configure
