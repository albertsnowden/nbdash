#!/usr/bin/env bash
#
# Install nbdash as a systemd service. Run as root on the target host.
#
# Usage:
#   sudo ./deploy/install.sh [path-to-binary]
#
# Defaults to dist/nbdash. Safe to re-run: it upgrades the binary and
# the unit in place and never overwrites your environment file.
#
set -euo pipefail

BIN_SRC="${1:-$(cd "$(dirname "$0")/.." && pwd)/dist/nbdash}"
UNIT_SRC="$(cd "$(dirname "$0")" && pwd)/nbdash.service"
ENV_SRC="$(cd "$(dirname "$0")" && pwd)/nbdash.env.example"

BIN_DST=/usr/local/bin/nbdash
UNIT_DST=/etc/systemd/system/nbdash.service
ENV_DIR=/etc/nbdash
ENV_DST="${ENV_DIR}/nbdash.env"

if [[ $EUID -ne 0 ]]; then
  echo "ERROR: run as root (sudo $0)" >&2
  exit 1
fi

for f in "$BIN_SRC" "$UNIT_SRC" "$ENV_SRC"; do
  if [[ ! -f "$f" ]]; then
    echo "ERROR: missing $f" >&2
    [[ "$f" == "$BIN_SRC" ]] && echo "       run ./build.sh first" >&2
    exit 1
  fi
done

# Replacing a running binary in place would fail with "Text file busy", so
# install to a temp name and rename over it — rename is atomic and leaves the
# running process on the old inode until it restarts.
echo "==> Installing binary to ${BIN_DST}"
install -m 0755 -o root -g root "$BIN_SRC" "${BIN_DST}.new"
mv -f "${BIN_DST}.new" "$BIN_DST"

echo "==> Installing unit to ${UNIT_DST}"
install -m 0644 -o root -g root "$UNIT_SRC" "$UNIT_DST"

# The environment file holds AUTH_CLIENT_SECRET, so it is root-only. systemd
# reads it as root before dropping privileges, so the service never needs it.
install -d -m 0700 -o root -g root "$ENV_DIR"

fresh_env=0
if [[ -f "$ENV_DST" ]]; then
  echo "==> Keeping existing ${ENV_DST}"
else
  echo "==> Installing template to ${ENV_DST}"
  install -m 0600 -o root -g root "$ENV_SRC" "$ENV_DST"
  fresh_env=1
fi

# Enforce ownership and permissions unconditionally, even when the file was
# already present and its content untouched above. AUTH_CLIENT_SECRET lives in
# this file, so the 0600 root:root invariant must hold after every run, not
# just the first one — a restored backup, a manual edit under a permissive
# umask, or an older version of this script could otherwise leave it readable
# and a re-run would silently accept that.
chown root:root "$ENV_DST"
chmod 0600 "$ENV_DST"

systemctl daemon-reload

# Starting with placeholders still in the file would just crash-loop, so stop
# here and let the operator fill it in first.
if [[ $fresh_env -eq 1 ]] || grep -q "replace-me" "$ENV_DST"; then
  cat <<EOF

Installed, not started — ${ENV_DST} still needs your settings.

  1. sudoedit ${ENV_DST}
  2. systemctl enable --now nbdash
  3. systemctl status nbdash
     journalctl -u nbdash -f

EOF
  exit 0
fi

echo "==> Restarting nbdash"
systemctl enable nbdash
systemctl restart nbdash
systemctl --no-pager --full status nbdash || true
