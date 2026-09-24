-- migrate:up transaction:false
PRAGMA journal_mode = WAL;

-- migrate:down
PRAGMA journal_mode = DELETE;
