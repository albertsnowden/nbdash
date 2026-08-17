#!/bin/sh
# Runs before dpkg removes the package's files.
#
# Only stop and disable the service on a genuine removal ($1 = remove). On an
# upgrade ($1 = upgrade) dpkg calls this right before unpacking the new
# version's files, and postinst restarts the service moments later — stopping
# it here would just cause a needless outage mid-upgrade.
set -e

case "$1" in
  remove)
    systemctl stop nbdash >/dev/null 2>&1 || true
    systemctl disable nbdash >/dev/null 2>&1 || true
    ;;
esac

exit 0
