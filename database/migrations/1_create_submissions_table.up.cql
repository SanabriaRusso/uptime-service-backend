CREATE TABLE IF NOT EXISTS submissions (
    // filled by uptime_service_backend
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
    // filled by zk-validator component
    state_hash TEXT, 
    parent TEXT,
    height INT,
    slot INT,
    validation_error TEXT,
    // was it verified by zk-validator
    verified BOOLEAN,
    PRIMARY KEY (submitted_at_date, submitted_at, submitter)
);
