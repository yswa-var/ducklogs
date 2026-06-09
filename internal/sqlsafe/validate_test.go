package sqlsafe

import "testing"

func TestValidateReadOnlySQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{name: "select is allowed", query: "SELECT * FROM app_logs LIMIT 10"},
		{name: "with is allowed", query: "WITH logs AS (SELECT * FROM app_logs) SELECT * FROM logs"},
		{name: "delete is blocked", query: "DELETE FROM app_logs", wantErr: true},
		{name: "copy is blocked inside select", query: "SELECT * FROM app_logs; COPY app_logs TO 'x.csv'", wantErr: true},
		{name: "non select is blocked", query: "PRAGMA table_info(app_logs)", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateReadOnlySQL(tt.query)
			if tt.wantErr && err == nil {
				t.Fatalf("ValidateReadOnlySQL(%q) expected error", tt.query)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateReadOnlySQL(%q) returned error: %v", tt.query, err)
			}
		})
	}
}
