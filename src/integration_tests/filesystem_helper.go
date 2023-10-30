package integration_tests

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return dstFile.Sync()
}

func emptyLocalFilesystemStorage(folderPath string) error {
	log.Printf("Emptying local filesystem folder: %s", folderPath)

	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		// Folder does not exist; nothing to do
		return nil
	}

	items, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}

	for _, item := range items {
		fullPath := folderPath + "/" + item.Name()
		if item.IsDir() {
			err = os.RemoveAll(fullPath)
		} else {
			err = os.Remove(fullPath)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func waitUntilLocalStorageHasBlocksAndSubmissions(directory string) error {
	log.Printf("Waiting for blocks and submissions to appear in the local directory: %s", directory)

	hasBlocks := false
	hasSubmissionsForToday := false
	currentDate := time.Now().Format("2006-01-02") // YYYY-MM-DD format

	timeout := time.After(TIMEOUT_IN_S * time.Second)
	tick := time.Tick(5 * time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("Timeout reached while waiting for local storage contents")
		case <-tick:
			items, err := os.ReadDir(directory)
			if err != nil {
				return err
			}
			// Reset the checks
			hasBlocks = false
			hasSubmissionsForToday = false

			if len(items) > 0 {
				log.Print("Found files in the local storage directory. Checking for blocks and submissions...")

				// Check for blocks
				blocksPath := filepath.Join(directory, "blocks")
				if items, err := os.ReadDir(blocksPath); err == nil {
					for _, item := range items {
						if strings.HasSuffix(item.Name(), ".dat") {
							hasBlocks = true
							break
						}
					}
				}

				// Check for submissions
				submissionsPathForToday := filepath.Join(directory, "submissions", currentDate)
				if items, err := os.ReadDir(submissionsPathForToday); err == nil {
					for _, item := range items {
						if strings.HasSuffix(item.Name(), ".json") && strings.Contains(item.Name(), currentDate) {
							hasSubmissionsForToday = true
							break
						}
					}
				}
			}

			// If both blocks and submissions for today are found, return
			if hasBlocks && hasSubmissionsForToday {
				log.Printf("Found blocks and submissions for today")
				return nil
			}
		}
	}
}
