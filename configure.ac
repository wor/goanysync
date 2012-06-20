#                                               -*- Autoconf -*-
# Process this file with autoconf to produce a configure script.

AC_PREREQ([2.69])
AC_INIT([goanysync], [1.0], [esa.maatta@iki.fi])

AC_CONFIG_SRCDIR(src/goanysync.go)

# Autoconf doesn't yet support "go" build tool,
# which is by default only go compiler installed on arch linux.
#_DEBUG_START_
#AC_PROG_GO([go])
#_DEBUG_END_

# Checks for programs.
AC_PROG_INSTALL
AC_PROG_SED

AC_MSG_CHECKING(for Arch Linux distribution (for rc.d srcipt installation))
if test -f "/etc/arch-release"; then
    AC_SUBST(is_archlinux, [yes])
    AC_MSG_RESULT(yes)
else
    AC_SUBST(is_archlinux, [no])
    AC_MSG_RESULT(no)
fi

AC_PATH_PROG([RSYNC_PATH], [rsync])


AC_CONFIG_FILES([Makefile:Makefile.in
                 conf/goanysync.cronjob:conf/goanysync.cronjob.in
                 conf/goanysync.conf:conf/goanysync.conf.in])

AC_OUTPUT
