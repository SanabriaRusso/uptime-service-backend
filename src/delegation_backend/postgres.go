package delegation_backend

import (
	"database/sql"
	"fmt"

	logging "github.com/ipfs/go-log/v2"
	_ "github.com/lib/pq"
)

type PostgreSQLContext struct {
	DB  *sql.DB
	Log *logging.ZapEventLogger
}

func NewPostgreSQL(cfg *PostgreSQLConfig) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func (ctx *PostgreSQLContext) insertSubmission(submission *Submission) error {
	query := `INSERT INTO submissions 
				(submitted_at_date, 
				 submitted_at, 
				 submitter, 
				 created_at, 
				 block_hash, 
				 remote_addr, 
				 peer_id, 
				 snark_work, 
				 graphql_control_port,
				 built_with_commit_sha)
			   VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := ctx.DB.Exec(query, submission.SubmittedAtDate, submission.SubmittedAt,
		submission.Submitter, submission.CreatedAt, submission.BlockHash,
		submission.RemoteAddr, submission.PeerId, submission.SnarkWork,
		submission.GraphqlControlPort, submission.BuiltWithCommitSha)
	return err
}

func (ctx *PostgreSQLContext) PostgreSQLSave(objs ObjectsToSave) {
	submissionToSave, err := objectToSaveToSubmission(objs, ctx.Log)
	if err != nil {
		ctx.Log.Errorf("PostgreSQLSave: Error preparing submission for saving: %v", err)
		return
	}

	if err := ctx.insertSubmission(submissionToSave); err != nil {
		ctx.Log.Errorf("PostgreSQLSave: Error saving submission to PostgreSQL: %v", err)
	} else {
		ctx.Log.Infof("PostgreSQLSave: Successfully saved submission for submitter: %v at %v", submissionToSave.Submitter, submissionToSave.SubmittedAt)
	}
}
