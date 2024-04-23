package delegation_backend

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type Submission struct {
	BlockHash          string    `json:"block_hash"`
	SubmittedAtDate    string    // Extracted from filePath
	SubmittedAt        time.Time // Extracted from filePath and parsed
	CreatedAt          time.Time `json:"created_at"`
	RemoteAddr         string    `json:"remote_addr"`
	PeerId             string    `json:"peer_id"`
	Submitter          string    `json:"submitter"` // is base58check-encoded submitter's public key
	RawBlock           []byte    `json:"raw_block,omitempty"`
	SnarkWork          []byte    `json:"snark_work,omitempty"`
	GraphqlControlPort int       `json:"graphql_control_port,omitempty"`
	BuiltWithCommitSha string    `json:"built_with_commit_sha,omitempty"`
}

type Block struct {
	BlockHash string
	RawBlock  []byte
}

type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

func parseSubmissionBytes(data []byte, filePath string) (*Submission, error) {
	// Extract information from filePath
	filePathParts := strings.Split(filePath, "/")
	if len(filePathParts) < 3 {
		return nil, fmt.Errorf("invalid file path: %s", filePath)
	}
	submittedAtDate := filePathParts[1]
	submittedAtWithSubmitter := strings.TrimSuffix(filePathParts[2], ".json")
	lastHyphenIndex := strings.LastIndex(submittedAtWithSubmitter, "-")
	submittedAtStr := submittedAtWithSubmitter[:lastHyphenIndex]

	// Parse submittedAtStr string into time.Time
	submittedAt, err := time.Parse("2006-01-02T15:04:05Z", submittedAtStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing submitted_at string: %w", err)
	}

	// Parse JSON contents
	var submission Submission
	err = json.Unmarshal(data, &submission)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling submission JSON: %w", err)
	}

	// Populate additional fields from filePath
	submission.SubmittedAtDate = submittedAtDate
	submission.SubmittedAt = submittedAt

	return &submission, nil
}

func parseBlockBytes(data []byte, filePath string) (*Block, error) {
	// Extract the filename without the extension to use as the BlockHash
	filename := filepath.Base(filePath)
	blockHash := strings.TrimSuffix(filename, filepath.Ext(filename))

	block := &Block{
		BlockHash: blockHash,
		RawBlock:  data,
	}
	return block, nil
}

func objectToSaveToSubmission(objs ObjectsToSave, logger Logger) (*Submission, error) {
	var submissionToSave *Submission = &Submission{}
	for path, bs := range objs {
		if strings.HasPrefix(path, "submissions/") {
			submission, err := parseSubmissionBytes(bs, path)
			if err != nil {
				logger.Errorf("Error parsing submission JSON: %v", err)
				continue
			}
			submissionToSave.BlockHash = submission.BlockHash
			submissionToSave.CreatedAt = submission.CreatedAt
			submissionToSave.GraphqlControlPort = submission.GraphqlControlPort
			submissionToSave.PeerId = submission.PeerId
			submissionToSave.RemoteAddr = submission.RemoteAddr
			submissionToSave.SnarkWork = submission.SnarkWork
			submissionToSave.SubmittedAt = submission.SubmittedAt
			submissionToSave.SubmittedAtDate = submission.SubmittedAtDate
			submissionToSave.Submitter = submission.Submitter
			submissionToSave.BuiltWithCommitSha = submission.BuiltWithCommitSha

		} else if strings.HasPrefix(path, "blocks/") {
			block, err := parseBlockBytes(bs, path)
			if err != nil {
				logger.Errorf("Error parsing block file: %v", err)
				continue
			}
			submissionToSave.RawBlock = block.RawBlock
			submissionToSave.BlockHash = block.BlockHash
		} else {
			logger.Errorf("Unknown path format: %s", path)
		}
	}

	if submissionToSave.Submitter == "" { // Check if a valid submission was processed
		return nil, fmt.Errorf("no valid submissions processed")
	}
	return submissionToSave, nil
}
