#!/bin/sh
# RPM %postun — runs after rpm removes the package's files.
#
# $1 = 0: the last (or only) installed version is gone — reload systemd so it
# notices the unit file is no longer on disk. $1 = 1+: an upgrade is still in
# progress (the new version's files are already unpacked), so there is
# nothing to do here — the new version's own %post already ran
# daemon-reload.
#
# Unlike deploy/deb/postrm.sh, there is no "purge" step to handle here at
# all: nbdash.env is packaged with %config(noreplace) (see the rpm
# override in nfpm.yaml), and RPM's own file-removal logic already decides
# per file whether to delete it (never modified from the packaged template —
# nothing an operator would miss) or rename it to nbdash.env.rpmsave and
# leave it (modified — holds real config, including the client secret).
# There is no dpkg-style "remove keeps it under its original name, purge
# deletes it" split to reimplement; RPM's default behaviour already gets
# both cases right without a script.
set -e

if [ "$1" -eq 0 ]; then
  systemctl daemon-reload >/dev/null 2>&1 || true
fi

exit 0
