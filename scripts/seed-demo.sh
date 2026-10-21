#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
docker compose --profile tools run --build --rm seed
