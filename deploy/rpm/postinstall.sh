#!/bin/sh
# RPM %post — runs after rpm unpacks the package's files, on both a fresh
# install ($1 = 1) and an upgrade ($1 = 2+). Unlike dpkg's postinst
# (deploy/deb/postinst.sh), which distinguishes those cases by string, RPM
# passes a count of how many versions of the package will be installed once
# this transaction finishes — but this script wants the same behaviour
# either way, so $1 is not branched on at all, matching what the deb script
# actually does inside its "configure" case.
set -e

ENV_DIR=/etc/nbdash
ENV_DST="${ENV_DIR}/nbdash.env"

# The environment file holds AUTH_CLIENT_SECRET, so it is root-only. systemd
# reads it as root before dropping privileges, so the service never needs it
# more permissive than this.
#
# Asserted unconditionally, not just "if we just created it": a restored
# backup, a manual edit under a permissive umask, or an older package
# version could otherwise leave it readable, and a re-run (or upgrade)
# would silently accept that. Same invariant deploy/deb/postinst.sh enforces.
if [ -f "$ENV_DST" ]; then
  chown root:root "$ENV_DST"
  chmod 0600 "$ENV_DST"
fi
chmod 0700 "$ENV_DIR" 2>/dev/null || true

systemctl daemon-reload >/dev/null 2>&1 || true

# Starting with placeholders still in the file would just crash-loop — same
# check deploy/deb/postinst.sh makes before its own enable/restart.
if [ ! -f "$ENV_DST" ] || grep -q "replace-me" "$ENV_DST" 2>/dev/null; then
  cat <<EOF

Installed, not started — ${ENV_DST} still needs your settings.

  1. sudoedit ${ENV_DST}
  2. systemctl enable --now nbdash
  3. systemctl status nbdash
     journalctl -u nbdash -f

EOF
else
  systemctl enable nbdash >/dev/null 2>&1 || true
  systemctl restart nbdash || true
  systemctl --no-pager --full status nbdash || true
fi

exit 0
