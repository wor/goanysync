#                                               -*- Autoconf -*-
# Process this file with autoconf to produce a configure script.

AC_PREREQ([2.68])
AC_INIT([goanysync], [1.03], [esa.maatta@iki.fi], [], [https://github.com/wor/goanysync])
AC_SUBST([fullname], ["Esa Määttä"])
AC_SUBST([C_YEARS], [2012])

AC_CONFIG_SRCDIR(src/goanysync/goanysync/goanysync.go)

# XXX: Autoconf doesn't yet support "go" build tool,
# which is by default only go compiler installed on Arch Linux.
#_DEBUG_START_
#AC_PROG_GO([go])
#_DEBUG_END_

# Checks for programs.
AC_PROG_INSTALL
AC_PROG_SED

# By default do not install tmpfiles.d config file,
# it's used only on systemd compatible distributions.
tmpfilesdir=

# Upstart job config file dir
initdir=

# Archlinux specific configuration
AC_MSG_CHECKING(for Arch Linux distribution (for rc.d srcipt installation))
if test -f "/etc/arch-release"; then
    # Set right prefix and sysconfdir if not given
    test "$prefix" = "NONE" && prefix=/usr
    test "$sysconfdir" = "\${prefix}/etc" && sysconfdir=/etc
    AC_SUBST([is_archlinux], [yes])
    tmpfilesdir="$libdir/tmpfiles.d"
    AC_MSG_RESULT(yes)
else
    AC_SUBST([is_archlinux], [no])
    AC_MSG_RESULT(no)
    # Test if etc tmpfiles.d dir exists
    if test -d "/etc/tmpfiles.d"; then
        sysconfdir=/etc
        tmpfilesdir="$sysconfdir/tmpfiles.d"
    fi
fi

# Test for upstart init dir
if test -d "/etc/init"; then
    sysconfdir=/etc
    initdir="$sysconfdir/init"
fi

AC_SUBST([tmpfilesdir])
AC_SUBST([initdir])

AC_PATH_PROG([RSYNC_PATH], [rsync], [])
if test "${ac_cv_path_RSYNC_PATH}x" == "x"; then
    AC_MSG_ERROR([Could not find rsync.])
fi

# For man page generation
AC_PATH_PROG([TXT2MAN_PATH], [txt2man], [])
AC_PATH_PROG([GZIP_PATH], [gzip], [])

# Makefile is full of bashism
AC_PATH_PROG([BASH_PATH], [bash], [])



AC_CONFIG_FILES([Makefile:Makefile.in
                 archlinux_pkg/PKGBUILD_dist:archlinux_pkg/PKGBUILD_dist.in
                 misc/license-gpl3-header:misc/license-gpl3-header.in
                 script/goanysync:script/goanysync.in
                 conf/goanysync.init.conf:conf/goanysync.init.conf.in
                 conf/goanysync.service:conf/goanysync.service.in
                 conf/goanysync.cronjob:conf/goanysync.cronjob.in
                 conf/goanysync.conf:conf/goanysync.conf.in])

AC_OUTPUT

echo "
$PACKAGE_NAME version $PACKAGE_VERSION
Prefix      : $prefix
Bindir      : $bindir
exec_prefix : $exec_prefix
mandir      : $mandir
Sysconfdir  : $sysconfdir
datarootdir : $datarootdir
datadir     : $datadir
Libdir      : $libdir
tmpfiles.d  : ${tmpfilesdir:-Not installed.}
initdir     : ${initdir:-Not installed.}
"
