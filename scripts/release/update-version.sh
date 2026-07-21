#!/bin/sh
set -eu

if [ -z "${MODREL_MODULE_DIR:-}" ]; then
	echo "MODREL_MODULE_DIR is required" >&2
	exit 1
fi
if [ -z "${MODREL_VERSION:-}" ]; then
	echo "MODREL_VERSION is required" >&2
	exit 1
fi
if [ "$#" -ne 1 ]; then
	echo "usage: update-version.sh <go-version-file>" >&2
	exit 1
fi

version_file="$MODREL_MODULE_DIR/$1"
if [ ! -f "$version_file" ]; then
	echo "version file not found: $version_file" >&2
	exit 1
fi
if ! grep -q '^const Version = "[^"]*"$' "$version_file"; then
	echo "Version constant not found in $version_file" >&2
	exit 1
fi

temporary_file="$version_file.tmp"
trap 'rm -f "$temporary_file"' EXIT
sed "s/^const Version = \"[^\"]*\"$/const Version = \"$MODREL_VERSION\"/" \
	"$version_file" > "$temporary_file"
mv "$temporary_file" "$version_file"
trap - EXIT
