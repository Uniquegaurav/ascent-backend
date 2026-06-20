#!/usr/bin/env bash
# Start the local Postgres (extracted from Postgres.app — no Docker needed).
set -e
PGBIN="$HOME/.local/Postgres.app/Contents/Versions/16/bin"
export PGDATA="$HOME/.local/summit-pgdata"

if [ ! -f "$PGDATA/PG_VERSION" ]; then
  echo "initializing Postgres data dir..."
  "$PGBIN/initdb" -A trust -U postgres -D "$PGDATA" >/tmp/initdb.log 2>&1
fi

if "$PGBIN/pg_ctl" -D "$PGDATA" status >/dev/null 2>&1; then
  echo "Postgres already running."
else
  "$PGBIN/pg_ctl" -D "$PGDATA" -l "$HOME/.local/summit-pg.log" -o "-p 5432 -k /tmp" -w start
fi

# Ensure the summit role + database exist.
"$PGBIN/psql" -h /tmp -p 5432 -U postgres -d postgres -tc \
  "SELECT 1 FROM pg_roles WHERE rolname='summit'" | grep -q 1 || \
  "$PGBIN/psql" -h /tmp -p 5432 -U postgres -d postgres -c \
  "CREATE ROLE summit SUPERUSER LOGIN PASSWORD 'summit';"
"$PGBIN/psql" -h /tmp -p 5432 -U postgres -d postgres -tc \
  "SELECT 1 FROM pg_database WHERE datname='summit'" | grep -q 1 || \
  "$PGBIN/psql" -h /tmp -p 5432 -U postgres -d postgres -c \
  "CREATE DATABASE summit OWNER summit;"
echo "Postgres ready on localhost:5432 (db=summit)."
