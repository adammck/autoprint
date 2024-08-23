package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInnerMain(t *testing.T) {
	t.Skip("not implemented")
}

func TestGetEtagPath(t *testing.T) {
	tests := []struct {
		url      string
		pattern  string
		expected string
	}{
		{"http://example.com", "etag-%s.txt", "etag-f0e6a6a97042a4f1f1c87f5f7d44315b2d852c2df5c7991cc66241bf7072d1c4.txt"},
		{"https://example.com/file", "etag-%s.txt", "etag-b14834559f5ee66a791f348bb05b37a925ab7875ac8229fc6c51ab670d2889a3.txt"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			actual := getEtagFilename(tt.pattern, tt.url)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestReadLastEtag(t *testing.T) {
	t.Skip("not implemented")
}

func TestWriteLastEtag(t *testing.T) {
	t.Skip("not implemented")
}

func TestDoRequest(t *testing.T) {
	t.Skip("not implemented")
}

func TestExtractFilename(t *testing.T) {
	tests := []struct {
		input        string
		expectOutput string
		expectError  bool
	}{
		{
			input:        ``,
			expectOutput: "",
			expectError:  false,
		},
		{
			input:        `invalid; 💩🧨`,
			expectOutput: "",
			expectError:  true,
		},
		{
			input:        `attachment; filename="example.txt"`,
			expectOutput: "example.txt",
			expectError:  false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			actual, err := extractFilename(tt.input)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectOutput, actual)
			}
		})
	}
}

func TestWriteOutput(t *testing.T) {
	t.Skip("not implemented")
}
