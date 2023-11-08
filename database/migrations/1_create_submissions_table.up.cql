CREATE TABLE IF NOT EXISTS submissions (
    submitted_at_date TEXT,
    submitted_at TIMESTAMP,
    submitter TEXT,
    created_at TIMESTAMP,
    block_hash TEXT,
    remote_addr TEXT,
    peer_id TEXT,
    snark_work BLOB,
    graphql_control_port INT,
    built_with_commit_sha TEXT,
    PRIMARY KEY (submitted_at_date, submitted_at, submitter)
);
