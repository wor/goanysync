#                                               -*- Autoconf -*-
# Process this file with autoconf to produce a configure script.

AC_PREREQ([2.69])
AC_INIT([goanysync], [1.0], [esa.maatta@iki.fi])

AC_CONFIG_SRCDIR(src/goanysync.go)

# XXX: Autoconf doesn't yet support "go" build tool,
# which is by default only go compiler installed on Arch Linux.
#_DEBUG_START_
#AC_PROG_GO([go])
#_DEBUG_END_

# Checks for programs.
AC_PROG_INSTALL
AC_PROG_SED

# Archlinux specific configuration
AC_MSG_CHECKING(for Arch Linux distribution (for rc.d srcipt installation))
if test -f "/etc/arch-release"; then
    # Set right prefix and sysconfdir if not given
    test "$prefix" = "NONE" && prefix=/usr
    test "$sysconfdir" = "\${prefix}/etc" && sysconfdir=/etc
    AC_SUBST([is_archlinux], [yes])
    AC_MSG_RESULT(yes)
else
    AC_SUBST([is_archlinux], [no])
    AC_MSG_RESULT(no)
fi

AC_PATH_PROG([RSYNC_PATH], [rsync], [])
if test "${ac_cv_path_RSYNC_PATH}x" == "x"; then
    AC_MSG_ERROR([Could not find rsync.])
fi

# For man page generation
AC_PATH_PROG([TXT2MAN_PATH], [txt2man], [])
AC_PATH_PROG([GZIP_PATH], [gzip], [])

# XXX: As AC_PROG_GO does not work, checking for go libs is not as straightforward.
#AC_SEARCH_LIBS([New], [goconfig], [have_goconfig=yes])

AC_CONFIG_FILES([Makefile:Makefile.in
                 README:README.in
                 conf/goanysync.cronjob:conf/goanysync.cronjob.in
                 conf/goanysync.conf:conf/goanysync.conf.in])

AC_OUTPUT
