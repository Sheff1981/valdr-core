#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 --app APP_PATH --output-dir DIR --arch arm64|amd64" >&2
}

app_path=""
output_dir=""
arch=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --app)
      app_path="${2:-}"
      shift 2
      ;;
    --output-dir)
      output_dir="${2:-}"
      shift 2
      ;;
    --arch)
      arch="${2:-}"
      shift 2
      ;;
    *)
      usage
      exit 2
      ;;
  esac
done

if [[ -z "$app_path" || -z "$output_dir" || -z "$arch" ]]; then
  usage
  exit 2
fi

case "$arch" in
  arm64)
    filename_arch="arm64"
    machine_pattern="arm64"
    ;;
  amd64)
    filename_arch="x64"
    machine_pattern="x86_64"
    ;;
  *)
    echo "unsupported macOS architecture: $arch" >&2
    exit 2
    ;;
esac

if [[ ! -d "$app_path" ]]; then
  echo "VALDR Desktop app bundle missing: $app_path" >&2
  exit 1
fi

app_path="$(cd "$(dirname "$app_path")" && pwd)/$(basename "$app_path")"
mkdir -p "$output_dir"
output_dir="$(cd "$output_dir" && pwd)"

desktop_bin="$app_path/Contents/MacOS/VALDR"
node_bin="$app_path/Contents/MacOS/valdrd"
miner_bin="$app_path/Contents/MacOS/valdr-miner"
for path in "$desktop_bin" "$node_bin" "$miner_bin"; do
  if [[ ! -x "$path" ]]; then
    echo "required executable missing: $path" >&2
    exit 1
  fi
done

version="$("$node_bin" version | awk '{print $NF}')"
if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+[-+A-Za-z0-9.]*$ ]]; then
  echo "invalid VALDR application version: $version" >&2
  exit 1
fi

for path in "$desktop_bin" "$node_bin" "$miner_bin"; do
  if ! file "$path" | grep -q "$machine_pattern"; then
    echo "unexpected architecture for $path; expected $machine_pattern" >&2
    file "$path" >&2
    exit 1
  fi
done

codesign --verify --deep --strict "$app_path"

dmg_name="VALDR-Desktop-$version-macos-$filename_arch.dmg"
dmg_out="$output_dir/$dmg_name"
work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT
stage_dir="$work_dir/image"
mkdir -p "$stage_dir"

ditto "$app_path" "$stage_dir/VALDR Desktop.app"
ln -s /Applications "$stage_dir/Applications"

create_dmg() {
  local attempt=1
  local max_attempts=3
  local log_file="$work_dir/hdiutil-create.log"

  while (( attempt <= max_attempts )); do
    rm -f "$dmg_out"
    : >"$log_file"

    if hdiutil create \
      -volname "VALDR Desktop" \
      -srcfolder "$stage_dir" \
      -ov \
      -format UDZO \
      "$dmg_out" >"$log_file" 2>&1; then
      return 0
    fi

    cat "$log_file" >&2
    rm -f "$dmg_out"

    if ! grep -q "Resource busy" "$log_file"; then
      echo "hdiutil create failed with a non-retryable error" >&2
      return 1
    fi

    if (( attempt == max_attempts )); then
      echo "hdiutil create remained resource-busy after $max_attempts attempts" >&2
      return 1
    fi

    sleep $(( attempt * 3 ))
    attempt=$(( attempt + 1 ))
  done
}

create_dmg

test -s "$dmg_out"
hdiutil verify "$dmg_out" >/dev/null
echo "built $dmg_out"
