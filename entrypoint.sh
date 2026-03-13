#!/bin/bash
set -e

# If Litestream env vars are set, use Litestream to manage the DB.
# Otherwise, run the server directly (local dev in Docker).
if [ -n "$LITESTREAM_ENDPOINT" ]; then
  # Restore the database from the replica if it exists.
  # -if-db-not-exists: skip restore if a local DB already exists (e.g. persistent disk).
  # -if-replica-exists: don't fail if this is the first deploy with no replica yet.
  litestream restore -if-db-not-exists -if-replica-exists -config /app/litestream.yml /app/data/sluff.db

  # Run the server under Litestream so WAL changes are continuously replicated.
  exec litestream replicate -exec "/app/sluff-server" -config /app/litestream.yml
else
  exec /app/sluff-server
fi
