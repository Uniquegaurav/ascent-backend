#!/usr/bin/env bash
# Start Postgres (if needed) then run the API in the foreground (Ctrl-C to stop).
set -e
DIR="$(cd "$(dirname "$0")/.." && pwd)"
export PATH="$HOME/.local/go-sdk/go/bin:$PATH"
bash "$DIR/scripts/db-start.sh"
cd "$DIR"
echo "starting API on :8080 ..."
go run ./cmd/api
