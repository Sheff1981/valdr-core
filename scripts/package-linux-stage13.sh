#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 --bin-dir DIR --output-dir DIR" >&2
}

bin_dir=""
output_dir=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --bin-dir)
      bin_dir="${2:-}"
      shift 2
      ;;
    --output-dir)
      output_dir="${2:-}"
      shift 2
      ;;
    *)
      usage
      exit 2
      ;;
  esac
done

if [[ -z "$bin_dir" || -z "$output_dir" ]]; then
  usage
  exit 2
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "$output_dir"
bin_dir="$(cd "$bin_dir" && pwd)"
output_dir="$(cd "$output_dir" && pwd)"

desktop_bin="$bin_dir/VALDR"
node_bin="$bin_dir/valdrd"
miner_bin="$bin_dir/valdr-miner"
desktop_entry="$repo_root/packaging/linux/valdr-desktop.desktop"
icon_source="$repo_root/apps/valdr-desktop/build/appicon.png"

for path in "$desktop_bin" "$node_bin" "$miner_bin"; do
  if [[ ! -x "$path" ]]; then
    echo "required executable missing: $path" >&2
    exit 1
  fi
done
for path in "$desktop_entry" "$icon_source"; do
  if [[ ! -f "$path" ]]; then
    echo "required packaging file missing: $path" >&2
    exit 1
  fi
done

version="$("$node_bin" version | awk '{print $NF}')"
if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+[-+A-Za-z0-9.]*$ ]]; then
  echo "invalid VALDR application version: $version" >&2
  exit 1
fi

appimage_name="VALDR-Desktop-$version-linux-x64.AppImage"
deb_name="VALDR-Desktop-$version-linux-amd64.deb"
appimage_out="$output_dir/$appimage_name"
deb_out="$output_dir/$deb_name"

work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

tools_dir="$work_dir/tools"
mkdir -p "$tools_dir"

linuxdeploy="$tools_dir/linuxdeploy-x86_64.AppImage"
appimagetool="$tools_dir/appimagetool-x86_64.AppImage"
runtime_file="$tools_dir/runtime-x86_64"

linuxdeploy_url="https://github.com/linuxdeploy/linuxdeploy/releases/download/1-alpha-20251107-1/linuxdeploy-x86_64.AppImage"
linuxdeploy_sha256="c20cd71e3a4e3b80c3483cef793cda3f4e990aca14014d23c544ca3ce1270b4d"
appimagetool_url="https://github.com/AppImage/appimagetool/releases/download/1.9.1/appimagetool-x86_64.AppImage"
appimagetool_sha256="ed4ce84f0d9caff66f50bcca6ff6f35aae54ce8135408b3fa33abfc3cb384eb0"
runtime_url="https://github.com/AppImage/type2-runtime/releases/download/20251108/runtime-x86_64"
runtime_sha256="2fca8b443c92510f1483a883f60061ad09b46b978b2631c807cd873a47ec260d"

download_verified() {
  local url="$1"
  local sha256="$2"
  local destination="$3"
  curl --fail --location --retry 3 --proto '=https' --tlsv1.2 "$url" --output "$destination"
  printf '%s  %s\n' "$sha256" "$destination" | sha256sum --check -
}

download_verified "$linuxdeploy_url" "$linuxdeploy_sha256" "$linuxdeploy"
download_verified "$appimagetool_url" "$appimagetool_sha256" "$appimagetool"
download_verified "$runtime_url" "$runtime_sha256" "$runtime_file"
chmod 0755 "$linuxdeploy" "$appimagetool" "$runtime_file"

appdir="$work_dir/VALDR.AppDir"
icon_file="$work_dir/valdr-desktop.png"
cp "$icon_source" "$icon_file"

APPIMAGE_EXTRACT_AND_RUN=1 "$linuxdeploy" \
  --appdir "$appdir" \
  --executable "$desktop_bin" \
  --desktop-file "$desktop_entry" \
  --icon-file "$icon_file"

install -Dm755 "$node_bin" "$appdir/usr/bin/valdrd"
install -Dm755 "$miner_bin" "$appdir/usr/bin/valdr-miner"
if [[ ! -x "$appdir/usr/bin/VALDR" ]]; then
  echo "linuxdeploy did not place VALDR in AppDir/usr/bin" >&2
  exit 1
fi
ln -sfn VALDR "$appdir/usr/bin/valdr-desktop"

if [[ ! -x "$appdir/AppRun" ]]; then
  echo "AppDir AppRun is missing" >&2
  exit 1
fi

ARCH=x86_64 VERSION="$version" APPIMAGE_EXTRACT_AND_RUN=1 "$appimagetool" \
  --runtime-file "$runtime_file" \
  "$appdir" \
  "$appimage_out"
chmod 0755 "$appimage_out"

deb_root="$work_dir/deb-root"
install -Dm755 "$desktop_bin" "$deb_root/usr/lib/valdr-desktop/VALDR"
install -Dm755 "$node_bin" "$deb_root/usr/lib/valdr-desktop/valdrd"
install -Dm755 "$miner_bin" "$deb_root/usr/lib/valdr-desktop/valdr-miner"
install -Dm644 "$desktop_entry" "$deb_root/usr/share/applications/valdr-desktop.desktop"
install -Dm644 "$icon_source" "$deb_root/usr/share/icons/hicolor/256x256/apps/valdr-desktop.png"
mkdir -p "$deb_root/usr/bin" "$deb_root/DEBIAN"

cat >"$deb_root/usr/bin/valdr-desktop" <<'EOF'
#!/bin/sh
exec /usr/lib/valdr-desktop/VALDR "$@"
EOF
chmod 0755 "$deb_root/usr/bin/valdr-desktop"

cat >"$deb_root/DEBIAN/control" <<EOF
Package: valdr-desktop
Version: $version
Section: utils
Priority: optional
Architecture: amd64
Maintainer: VALDR
Depends: libgtk-3-0 | libgtk-3-0t64, libwebkit2gtk-4.1-0
Description: VALDR Desktop Testnet wallet and full node
 VALDR Desktop bundles the VALDR Testnet wallet UI, local full node and miner.
EOF

dpkg-deb --root-owner-group --build "$deb_root" "$deb_out"

test -s "$appimage_out"
test -s "$deb_out"

echo "built $appimage_out"
echo "built $deb_out"
