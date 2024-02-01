package delegation_backend

import (
	"errors"
	"testing"
	"time"
)

func TestExponentialBackoff(t *testing.T) {

	var mockOperationCalls int

	// Test cases
	testCases := []struct {
		name                   string
		operation              Operation
		expectedOperationCalls int
		shouldError            bool
	}{
		{
			name: "TC 1: Success on first attempt",
			operation: func() error {
				mockOperationCalls++
				return nil // Success on first attempt
			},
			expectedOperationCalls: 1,
			shouldError:            false,
		},
		{
			name: "TC 2: Success on second attempt",
			operation: func() error {
				mockOperationCalls++
				if mockOperationCalls < 2 {
					return errors.New("temporary error")
				}
				return nil // Success on second attempt
			},
			expectedOperationCalls: 2,
			shouldError:            false,
		},
		{
			name: "TC 3: Fails after 3 attempts",
			operation: func() error {
				mockOperationCalls++
				return errors.New("persistent error")
			},
			expectedOperationCalls: 3,
			shouldError:            true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset operation call count
			mockOperationCalls = 0

			// Run the ExponentialBackoff function
			err := ExponentialBackoff(tc.operation, 3, 10*time.Millisecond)

			// Check if error matches expectation
			if (err != nil) != tc.shouldError {
				t.Errorf("%s: Expected error: %v, got: %v", tc.name, tc.shouldError, err != nil)
			}

			// Assertions for operation call count
			if mockOperationCalls != tc.expectedOperationCalls {
				t.Errorf("%s: Expected %d operation calls, got %d", tc.name, tc.expectedOperationCalls, mockOperationCalls)
			}
		})
	}
}
