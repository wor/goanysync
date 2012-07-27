goanysync
=========

goanysync is a relatively small program to replace given directories in HDD/SSD
with symlinks to tmpfs and to sync this tmpfs contents back to HDD/SSD. It is a
rewrite of "anything-sync-daemon" with go programming language
(see:[Anything-sync-daemon](https://wiki.archlinux.org/index.php/Anything-sync-daemon)).

Two main use cases are reducing wear on SSD and speeding up programs by moving
their data directories to tmpfs.


Motivation
----------

goanysync began as fork of anything-sync-daemon (by graysky), but is now
basically a complete rewrite and only the documentation and functionality still
bares resemblance to asd. Rewrote was mainly inspired by permission problems
with symlinked dirs and by the original programs bash code which, for example,
contained this line: [[ -d "$VOLATILE$i" ]] || mkdir -p "$VOLATILE$i" ||
"install -Dm755 $VOLATILE$i"


Run dependencies
----------------

* rsync


Build dependencies
------------------

* autoconf
* automake
* libtool
* go (golang)
* gzip
* txt2man


Build and install (git)
-----------------------

    ./autogen.sh
    make
    make install

Alternatively for Arch Linux an aur package is provided:
[https://aur.archlinux.org/packages.php?ID=60715](https://aur.archlinux.org/packages.php?ID=60715)


Build and install (source package)
----------------------------------

    ./configure
    make
    make install

Source package for the most recent tagged version is located at
[goanysync-1.0.tar.gz](https://github.com/downloads/wor/goanysync/goanysync-1.0.tar.gz)


Debian package
--------------

Also a Debian package is provided for the most recent tagged version:
[goanysync_1.0-1_i386.deb](https://github.com/downloads/wor/goanysync/goanysync_1.0-1_i386.deb)

The package was build on Ubuntu 12.04. Recent .deb package can always be build from
goanysync git source using commands:

    ./autogen.sh
    make deb

The automatically generated package definitely is not up to packaging standards
but should be good enough for testing.


Usage
-----

Just edit installed (default location) /etc/goanysync.conf to suit your needs
and call:

    goanysync start

And remember to call:

    goanysync stop

Before booting.

Daemon scripts to do above automatically are provided for Archlinux rc.d,
systemd and upstart systems.
