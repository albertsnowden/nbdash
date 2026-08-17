#!/bin/sh
# Runs after dpkg removes the package's files.
#
# $1 = remove: dpkg has already deleted the binary and the unit file (they
# are ordinary packaged files, not conffiles). nbdash.env is a conffile
# (config|noreplace in nfpm.yaml) so dpkg leaves it behind on a plain
# remove — that is deliberate: it holds AUTH_CLIENT_SECRET and any other
# tuning the operator did, and "remove" should be safe to follow with a
# reinstall that picks the same config back up.
#
# $1 = purge: the operator explicitly asked for the config to go too
# (dpkg -P / apt purge). This is the only place that ever deletes
# /etc/nbdash, and it only runs on an explicit purge, never a
# plain remove or an upgrade.
#
# There is no system user to clean up either way — the unit runs under
# DynamicUser=yes, a transient UID systemd allocates and releases itself,
# never a real account this script would need to userdel.
set -e

case "$1" in
  remove)
    systemctl daemon-reload >/dev/null 2>&1 || true
    ;;
  purge)
    rm -rf /etc/nbdash
    systemctl daemon-reload >/dev/null 2>&1 || true
    ;;
esac

exit 0
