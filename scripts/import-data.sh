#!/bin/sh
set -eu
if [ "$#" -ne 1 ] || [ ! -f "$1" ]; then
  echo "Usage: $0 /path/to/vuc.json" >&2
  exit 1
fi
source_file="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
cd "$(dirname "$0")/.."
# The source is read-only. Import refuses a nonempty target database.
docker compose --profile tools run --build --rm \
  --volume "$source_file:/import/vuc.json:ro" importdata
