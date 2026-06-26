package model

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDate_UnmarshalGQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   interface{}
		want    string // expected date string in YYYY-MM-DD
		wantErr string
	}{
		{
			name:  "valid date string",
			input: "2026-06-16",
			want:  "2026-06-16",
		},
		{
			name:    "non-string input (int)",
			input:   20260616,
			wantErr: "Date must be a string",
		},
		{
			name:    "non-string input (nil)",
			input:   nil,
			wantErr: "Date must be a string",
		},
		{
			name:    "invalid date format",
			input:   "16-06-2026",
			wantErr: "Date must be in YYYY-MM-DD format",
		},
		{
			name:    "invalid date value",
			input:   "2026-13-01",
			wantErr: "Date must be in YYYY-MM-DD format",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: "Date must be in YYYY-MM-DD format",
		},
		{
			name:    "partial date",
			input:   "2026-06",
			wantErr: "Date must be in YYYY-MM-DD format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var d Date
			err := d.UnmarshalGQL(tt.input)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, d.Time.Format("2006-01-02"))
		})
	}
}

func TestDate_MarshalGQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		date Date
		want string
	}{
		{
			name: "standard date",
			date: Date{Time: time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC)},
			want: `"2026-06-16"`,
		},
		{
			name: "leap year date",
			date: Date{Time: time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)},
			want: `"2024-02-29"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			tt.date.MarshalGQL(&buf)
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestMarshalDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input time.Time
		want  string
	}{
		{
			name:  "standard date",
			input: time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC),
			want:  `"2026-06-16"`,
		},
		{
			name:  "new years day",
			input: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			want:  `"2026-01-01"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			marshaler := MarshalDate(tt.input)
			var buf bytes.Buffer
			marshaler.MarshalGQL(&buf)
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestUnmarshalDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   interface{}
		want    string
		wantErr string
	}{
		{
			name:  "valid date string",
			input: "2026-06-16",
			want:  "2026-06-16",
		},
		{
			name:    "non-string input",
			input:   42,
			wantErr: "Date must be a string",
		},
		{
			name:    "invalid format",
			input:   "06/16/2026",
			wantErr: "Date must be in YYYY-MM-DD format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := UnmarshalDate(tt.input)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, result.Format("2006-01-02"))
		})
	}
}

func TestDate_String(t *testing.T) {
	t.Parallel()

	d := Date{Time: time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC)}
	assert.Equal(t, "2026-06-16", d.Time.Format("2006-01-02"))
}

func TestDate_UnmarshalGQL_ResetsPreviousValue(t *testing.T) {
	t.Parallel()

	d := Date{Time: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
	err := d.UnmarshalGQL("2026-12-31")
	require.NoError(t, err)
	assert.Equal(t, "2026-12-31", d.Time.Format("2006-01-02"))
}

func TestDate_MarshalGQL_Writer(t *testing.T) {
	t.Parallel()

	d := Date{Time: time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC)}
	var buf bytes.Buffer
	d.MarshalGQL(&buf)
	assert.Equal(t, `"2026-06-16"`, buf.String())
	assert.True(t, strings.HasPrefix(buf.String(), `"`), "should be quoted")
	assert.True(t, strings.HasSuffix(buf.String(), `"`), "should be quoted")
}
