package ingest

import (
	"testing"
	"time"
)

func TestNextRunTime(t *testing.T) {
	loc := time.Local

	tests := []struct {
		name    string
		spec    string
		now     time.Time
		want    time.Time
		wantErr bool
	}{
		{
			name: "after target hour → tomorrow",
			spec: "0 3 * * *",
			now:  time.Date(2026, 8, 5, 10, 0, 0, 0, loc),
			want: time.Date(2026, 8, 6, 3, 0, 0, 0, loc),
		},
		{
			name: "before target hour → today",
			spec: "0 3 * * *",
			now:  time.Date(2026, 8, 5, 2, 0, 0, 0, loc),
			want: time.Date(2026, 8, 5, 3, 0, 0, 0, loc),
		},
		{
			name: "after target hour (same hour, later minutes) → tomorrow",
			spec: "0 3 * * *",
			now:  time.Date(2026, 8, 5, 3, 30, 0, 0, loc),
			want: time.Date(2026, 8, 6, 3, 0, 0, 0, loc),
		},
		{
			name:    "invalid spec",
			spec:    "bad",
			now:     time.Date(2026, 8, 5, 12, 0, 0, 0, loc),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextRunTime(tt.spec, tt.now)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NextRunTime(%q, %v) = %v, want error", tt.spec, tt.now, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NextRunTime(%q, %v) unexpected error: %v", tt.spec, tt.now, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("NextRunTime(%q, %v) = %v, want %v", tt.spec, tt.now, got, tt.want)
			}
		})
	}
}
