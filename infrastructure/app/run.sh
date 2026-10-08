#!/bin/bash

set -e

rm -f /app/app.db

litestream restore -if-replica-exists -config /etc/litestream.yaml /app/app.db
sqlite3def --file=/app/schema.sql /app/app.db --apply --enable-drop --config-inline 'skip_tables: ^_litestream_.*$'
exec litestream replicate -exec /app/app -config /etc/litestream.yaml
