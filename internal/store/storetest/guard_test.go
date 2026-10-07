package storetest

import "testing"

func TestCheckNotDemo(t *testing.T) {
	tests := []struct {
		name    string
		current string
		wantErr bool
	}{
		{name: "demo database refused", current: DemoDatabase, wantErr: true},
		{name: "per-run test database allowed", current: "outlet_owl_test_123_0a1b2c3d", wantErr: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckNotDemo(tc.current)
			if (err != nil) != tc.wantErr {
				t.Fatalf("CheckNotDemo(%q) error = %v, want error %v", tc.current, err, tc.wantErr)
			}
		})
	}
}
