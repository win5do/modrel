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

printf '%s\n' "$MODREL_VERSION" > "$MODREL_MODULE_DIR/VERSION"
