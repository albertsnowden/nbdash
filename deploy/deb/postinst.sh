#!/bin/sh
# Runs after dpkg unpacks the package's files — on both a fresh install and an
# upgrade. dpkg has already placed the binary, the systemd unit, and (on a
# fresh install only — see nfpm.yaml's config|noreplace on that entry) the
# nbdash.env template; this script's job is everything dpkg itself can't
# express: permissions that must hold regardless of how the file got there,
# and deciding whether it's safe to actually start the service.
set -e

ENV_DIR=/etc/nbdash
ENV_DST="${ENV_DIR}/nbdash.env"

case "$1" in
  configure)
    # The environment file holds AUTH_CLIENT_SECRET, so it is root-only.
    # systemd reads it as root before dropping privileges, so the service
    # never needs it more permissive than this.
    #
    # Asserted unconditionally, not just "if we just created it": a restored
    # backup, a manual edit under a permissive umask, or an older package
    # version could otherwise leave it readable, and a re-run (or upgrade)
    # would silently accept that. Same invariant install.sh enforces.
    if [ -f "$ENV_DST" ]; then
      chown root:root "$ENV_DST"
      chmod 0600 "$ENV_DST"
    fi
    chmod 0700 "$ENV_DIR" 2>/dev/null || true

    systemctl daemon-reload >/dev/null 2>&1 || true

    # Starting with placeholders still in the file would just crash-loop —
    # same check install.sh makes before its own enable/restart.
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
    ;;
esac

exit 0
