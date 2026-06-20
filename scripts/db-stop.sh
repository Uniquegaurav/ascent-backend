#!/usr/bin/env bash
# Stop the local Postgres.
PGBIN="$HOME/.local/Postgres.app/Contents/Versions/16/bin"
export PGDATA="$HOME/.local/summit-pgdata"
"$PGBIN/pg_ctl" -D "$PGDATA" -m fast stop
