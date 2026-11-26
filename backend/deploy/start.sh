#!/bin/sh
# Container entry point: migrate the database, then start the API.
# Render's free plan has no pre-deploy step, so migrations run here, as in the
# School Management System. Migrations must stay backward compatible so a
# rollback to the previous release still works (see RELEASING.md).
#
# Configuration comes from environment variables only (no config.yaml):
#   TD_DATABASE_URL      Supabase session-pooler connection string (required)
#   TD_ALLOWED_ORIGINS   frontend origins, comma-separated (required)
#   TD_TRUSTED_PROXIES   the host's proxy range, e.g. 10.0.0.0/8 on Render
#   PORT                 set by the host
set -eu

echo "==> Migrating the database"
# The database may still be waking up (Supabase free tier): retry for a minute.
tries=0
until /app/ticketdoctor migrate; do
  tries=$((tries + 1))
  if [ "$tries" -ge 12 ]; then
    echo "    Migration failed 12 times; giving up"
    exit 1
  fi
  echo "    Database not ready; retrying in 5 seconds"
  sleep 5
done

echo "==> Starting the API"
exec /app/ticketdoctor server
