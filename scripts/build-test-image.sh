#!/bin/sh
set -eu
context=$(mktemp -d)
trap 'rm -rf "$context"' EXIT HUP INT TERM
python3 scripts/release-context.py --test-snapshot --output "$context/source"
docker build -t mediagrap:dev "$context/source"
