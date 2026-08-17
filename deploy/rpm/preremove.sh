#!/bin/sh
# RPM %preun — runs before rpm removes the package's files.
#
# $1 = 0 means this is the last (or only) installed version going away: a
# genuine uninstall. $1 = 1+ means a newer version is about to replace it as
# part of an upgrade transaction — stopping the service here would just be a
# needless outage moments before the new version's %post (postinstall.sh)
# restarts it. Same distinction deploy/deb/prerm.sh makes between dpkg's
# "remove" and "upgrade" calls, expressed with RPM's numeric convention
# instead of Debian's string one.
set -e

if [ "$1" -eq 0 ]; then
  systemctl stop nbdash >/dev/null 2>&1 || true
  systemctl disable nbdash >/dev/null 2>&1 || true
fi

exit 0
