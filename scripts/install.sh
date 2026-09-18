#!/bin/sh

set -eu

repo="Eliran-Turgeman/repear"
install_dir="${REAPER_INSTALL_DIR:-}"
requested_version="${REAPER_VERSION:-latest}"

require() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "reaper installer: required command not found: $1" >&2
    exit 1
  fi
}

require curl
require uname

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *)
    echo "reaper installer: unsupported operating system: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *)
    echo "reaper installer: unsupported CPU architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

if [ "$requested_version" = "latest" ]; then
  latest_url="$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest")"
  tag="${latest_url##*/}"
else
  tag="$requested_version"
  case "$tag" in
    v*) ;;
    *) tag="v$tag" ;;
  esac
fi

version="${tag#v}"
if [ -z "$version" ] || [ "$version" = "latest" ]; then
  echo "reaper installer: could not determine the release version" >&2
  exit 1
fi

asset="reaper_${version}_${os}_${arch}.tar.gz"
base_url="https://github.com/$repo/releases/download/$tag"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

echo "Downloading Reaper $version for $os/$arch..."
curl -fsSL "$base_url/$asset" -o "$tmp_dir/$asset"
curl -fsSL "$base_url/checksums.txt" -o "$tmp_dir/checksums.txt"

expected="$(awk -v asset="$asset" '$2 == asset { print $1 }' "$tmp_dir/checksums.txt")"
if [ -z "$expected" ]; then
  echo "reaper installer: $asset is missing from checksums.txt" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp_dir/$asset" | awk '{ print $1 }')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$tmp_dir/$asset" | awk '{ print $1 }')"
else
  echo "reaper installer: sha256sum or shasum is required to verify the download" >&2
  exit 1
fi

if [ "$actual" != "$expected" ]; then
  echo "reaper installer: checksum verification failed" >&2
  exit 1
fi

tar -xzf "$tmp_dir/$asset" -C "$tmp_dir"

if [ -z "$install_dir" ]; then
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    install_dir="/usr/local/bin"
  else
    install_dir="$HOME/.local/bin"
  fi
fi

mkdir -p "$install_dir"
install -m 0755 "$tmp_dir/reaper" "$install_dir/reaper"

echo "Installed Reaper $version to $install_dir/reaper"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) echo "Add $install_dir to PATH to run reaper from any directory." ;;
esac
