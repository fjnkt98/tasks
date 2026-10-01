#!/bin/bash

set -e

rm -f /app/app.db

litestream restore -if-replica-exists -config /etc/litestream.yaml /app/app.db
litestream replicate -exec '/app/app --migrate' -config /etc/litestream.yaml
