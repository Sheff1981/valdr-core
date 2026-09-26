#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 --deb PATH" >&2
}

deb=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --deb)
      deb="${2:-}"
      shift 2
      ;;
    *)
      usage
      exit 2
      ;;
  esac
done

if [[ -z "$deb" || ! -f "$deb" ]]; then
  usage
  exit 2
fi

deb="$(cd "$(dirname "$deb")" && pwd)/$(basename "$deb")"
work="$(mktemp -d)"
test_home="$(mktemp -d)"
cleanup() {
  sudo dpkg -r valdr-desktop >/dev/null 2>&1 || true
  rm -rf "$work" "$test_home"
}
trap cleanup EXIT

base_version="$(dpkg-deb -f "$deb" Version)"
old_version="${base_version}+stage18.1"
new_version="${base_version}+stage18.2"

old_root="$work/old-root"
new_root="$work/new-root"
dpkg-deb -R "$deb" "$old_root"
dpkg-deb -R "$deb" "$new_root"
sed -i "s/^Version:.*/Version: $old_version/" "$old_root/DEBIAN/control"
sed -i "s/^Version:.*/Version: $new_version/" "$new_root/DEBIAN/control"

old_deb="$work/valdr-desktop-stage18-old.deb"
new_deb="$work/valdr-desktop-stage18-new.deb"
dpkg-deb --root-owner-group --build "$old_root" "$old_deb" >/dev/null
dpkg-deb --root-owner-group --build "$new_root" "$new_deb" >/dev/null

export HOME="$test_home"
export XDG_DATA_HOME="$HOME/.local/share"
data_root="$XDG_DATA_HOME/valdr"
mkdir -p "$data_root"
printf 'stage18-preserve\n' >"$data_root/release-upgrade-rollback.marker"

sudo dpkg -i "$old_deb" >/dev/null
test "$(dpkg-query -W -f='${Version}' valdr-desktop)" = "$old_version"
test -x /usr/lib/valdr-desktop/VALDR
test -x /usr/lib/valdr-desktop/valdrd
test -x /usr/lib/valdr-desktop/valdr-miner
/usr/lib/valdr-desktop/valdrd version >/dev/null
test -f "$data_root/release-upgrade-rollback.marker"

# Simulated release upgrade: package-manager version changes while the package
# contract and VALDR user-data directory remain the same.
sudo dpkg -i "$new_deb" >/dev/null
test "$(dpkg-query -W -f='${Version}' valdr-desktop)" = "$new_version"
test -f "$data_root/release-upgrade-rollback.marker"
/usr/lib/valdr-desktop/valdrd version >/dev/null

# Simulated rollback to the previous package image. The rollback may replace
# program files, but it must not delete wallet/node/user data.
sudo dpkg -i --force-downgrade "$old_deb" >/dev/null
test "$(dpkg-query -W -f='${Version}' valdr-desktop)" = "$old_version"
test -f "$data_root/release-upgrade-rollback.marker"
/usr/lib/valdr-desktop/valdrd version >/dev/null

sudo dpkg -r valdr-desktop >/dev/null
test ! -e /usr/lib/valdr-desktop
test ! -e /usr/bin/valdr-desktop
test -f "$data_root/release-upgrade-rollback.marker"

echo "Stage 18 Linux release upgrade/rollback package contract: PASS"
