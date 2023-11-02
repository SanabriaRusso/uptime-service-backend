CREATE TABLE IF NOT EXISTS submissions (
    submitted_at_date TEXT,
    submitted_at TIMESTAMP,
    submitter TEXT,
    remote_addr TEXT,
    peer_id TEXT,
    snark_work TEXT,
    block_hash TEXT,
    PRIMARY KEY (submitted_at_date, submitted_at, submitter)
);
