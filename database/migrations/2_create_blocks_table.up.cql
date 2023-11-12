CREATE TABLE IF NOT EXISTS blocks (
    block_hash TEXT PRIMARY KEY,
    raw_block BLOB
);