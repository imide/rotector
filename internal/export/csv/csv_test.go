package csv

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/robalyx/rotector/internal/export/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// verifyCSVFile reads a CSV file and verifies its contents match the expected records.
func verifyCSVFile(t *testing.T, filepath string, expectedRecords []*types.ExportRecord) {
	// Open file
	file, err := os.Open(filepath)
	require.NoError(t, err)
	defer file.Close()

	// Create CSV reader
	reader := csv.NewReader(file)

	// Read and verify header
	header, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, []string{"hash", "status", "reason"}, header)

	// Read and verify each record
	for _, expected := range expectedRecords {
		record, err := reader.Read()
		require.NoError(t, err)
		assert.Equal(t, expected.Hash, record[0])
		assert.Equal(t, expected.Status, record[1])
		assert.Equal(t, expected.Reason, record[2])
	}

	// Verify we're at the end
	_, err = reader.Read()
	assert.Equal(t, io.EOF, err, "expected EOF after last record")
}

func TestExporter_Export(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name         string
		userRecords  []*types.ExportRecord
		groupRecords []*types.ExportRecord
		wantErr      bool
	}{
		{
			name: "basic export",
			userRecords: []*types.ExportRecord{
				{
					Hash:   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
					Status: "confirmed",
					Reason: "test reason",
				},
				{
					Hash:   "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
					Status: "flagged",
					Reason: "another reason",
				},
			},
			groupRecords: []*types.ExportRecord{
				{
					Hash:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1",
					Status: "flagged",
					Reason: "group test reason",
				},
				{
					Hash:   "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb2",
					Status: "confirmed",
					Reason: "another group reason",
				},
			},
			wantErr: false,
		},
		{
			name:         "empty records",
			userRecords:  []*types.ExportRecord{},
			groupRecords: []*types.ExportRecord{},
			wantErr:      false,
		},
		{
			name: "records with special characters",
			userRecords: []*types.ExportRecord{
				{
					Hash:   "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc3",
					Status: "confirmed",
					Reason: "reason with, comma",
				},
				{
					Hash:   "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd4",
					Status: "flagged",
					Reason: "reason with \"quotes\"",
				},
			},
			groupRecords: []*types.ExportRecord{},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new exporter
			e := New(tempDir)

			// Perform export
			err := e.Export(tt.userRecords, tt.groupRecords)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify users.csv
			if len(tt.userRecords) > 0 {
				verifyCSVFile(t, filepath.Join(tempDir, "users.csv"), tt.userRecords)
			}

			// Verify groups.csv
			if len(tt.groupRecords) > 0 {
				verifyCSVFile(t, filepath.Join(tempDir, "groups.csv"), tt.groupRecords)
			}
		})
	}
}

func TestExporter_ExistingFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create existing files
	files := []string{"users.csv", "groups.csv"}
	for _, file := range files {
		err := os.WriteFile(filepath.Join(tempDir, file), []byte("existing content"), 0o644)
		require.NoError(t, err)
	}

	e := New(tempDir)

	records := []*types.ExportRecord{
		{
			Hash:   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			Status: "confirmed",
			Reason: "test reason",
		},
	}

	// Export should overwrite existing files
	err := e.Export(records, records)
	require.NoError(t, err)

	// Verify both files were overwritten
	verifyCSVFile(t, filepath.Join(tempDir, "users.csv"), records)
	verifyCSVFile(t, filepath.Join(tempDir, "groups.csv"), records)
}
