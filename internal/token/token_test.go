package token

import "testing"

func TestRefreshTokenFromServerByRefreshCode(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name: "refreskToken",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RefreshTokenFromServerByRefreshCode(); (err != nil) != tt.wantErr {
				t.Errorf("RefreshTokenFromServerByRefreshCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
