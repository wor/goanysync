Changelog
=========

v1.10 ()
-----

- Commands start and stop do additional sanity checks regarding TMPFS dir.
Errors are reported about extra volatile directories not defined in the config
files WHATTOSYNC list. This should give an error if paths were removed from the
WHATTOSYNC list between start and stop commands.

- New "info" command gives information about synced paths and extra (probably)
leftover paths in the TMPFS dir.

- Lockfile is defined in the config file, meaning the program can now be more
easily made to run as any user.

- Rsync errors are now logged. Now it's easier to notice out-of-space errors for
example.

- Volatile paths are removed after stop/unsync. This saves space/memory. Also
it's easier to see what is synced and what not.

- Added more informative logging and console verbose log switch '-v'. Also other
log related fixes.

- Fixed issue 7: [issue 7](https://github.com/wor/goanysync/issues/7)

v1.02 (2012-07-30)
-----

- Small fixes.

v1.01 (2012-07-30)
-----

- Small fixes.

v1.0 (2012-07-27)
-----

- First release
