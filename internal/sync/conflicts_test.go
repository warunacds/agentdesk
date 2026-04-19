package sync

import (
	"testing"
)

// ---------------------------------------------------------------------------
// CompareVersions
// ---------------------------------------------------------------------------

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name   string
		local  map[string]uint64
		remote map[string]uint64
		want   VersionOrder
	}{
		{
			name:   "local newer single key",
			local:  map[string]uint64{"A": 3, "B": 2},
			remote: map[string]uint64{"A": 2, "B": 2},
			want:   LocalNewer,
		},
		{
			name:   "remote newer single key",
			local:  map[string]uint64{"A": 2, "B": 2},
			remote: map[string]uint64{"A": 3, "B": 2},
			want:   RemoteNewer,
		},
		{
			name:   "concurrent divergent keys",
			local:  map[string]uint64{"A": 3, "B": 2},
			remote: map[string]uint64{"A": 2, "B": 3},
			want:   Concurrent,
		},
		{
			name:   "equal identical clocks",
			local:  map[string]uint64{"A": 2, "B": 2},
			remote: map[string]uint64{"A": 2, "B": 2},
			want:   Equal,
		},
		{
			name:   "empty local vs filled remote",
			local:  map[string]uint64{},
			remote: map[string]uint64{"A": 1},
			want:   RemoteNewer,
		},
		{
			name:   "filled local vs empty remote",
			local:  map[string]uint64{"A": 1},
			remote: map[string]uint64{},
			want:   LocalNewer,
		},
		{
			name:   "both empty",
			local:  map[string]uint64{},
			remote: map[string]uint64{},
			want:   Equal,
		},
		{
			name:   "local newer all keys greater or equal",
			local:  map[string]uint64{"A": 5, "B": 3, "C": 1},
			remote: map[string]uint64{"A": 4, "B": 3, "C": 0},
			want:   LocalNewer,
		},
		{
			name:   "remote newer all keys greater or equal",
			local:  map[string]uint64{"A": 1, "B": 1},
			remote: map[string]uint64{"A": 2, "B": 2},
			want:   RemoteNewer,
		},
		{
			name:   "concurrent with disjoint keys",
			local:  map[string]uint64{"A": 1},
			remote: map[string]uint64{"B": 1},
			want:   Concurrent,
		},
		{
			name:   "local has extra key remote does not",
			local:  map[string]uint64{"A": 2, "B": 2, "C": 1},
			remote: map[string]uint64{"A": 2, "B": 2},
			want:   LocalNewer,
		},
		{
			name:   "remote has extra key local does not",
			local:  map[string]uint64{"A": 2, "B": 2},
			remote: map[string]uint64{"A": 2, "B": 2, "C": 1},
			want:   RemoteNewer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareVersions(tt.local, tt.remote)
			if got != tt.want {
				t.Errorf("CompareVersions(%v, %v) = %d, want %d", tt.local, tt.remote, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// MergeVersions
// ---------------------------------------------------------------------------

func TestMergeVersions(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]uint64
		b    map[string]uint64
		want map[string]uint64
	}{
		{
			name: "takes max of each key",
			a:    map[string]uint64{"A": 3, "B": 1},
			b:    map[string]uint64{"A": 1, "B": 4},
			want: map[string]uint64{"A": 3, "B": 4},
		},
		{
			name: "keys only in a",
			a:    map[string]uint64{"A": 5, "C": 2},
			b:    map[string]uint64{"A": 3},
			want: map[string]uint64{"A": 5, "C": 2},
		},
		{
			name: "keys only in b",
			a:    map[string]uint64{"A": 1},
			b:    map[string]uint64{"A": 2, "D": 7},
			want: map[string]uint64{"A": 2, "D": 7},
		},
		{
			name: "both empty",
			a:    map[string]uint64{},
			b:    map[string]uint64{},
			want: map[string]uint64{},
		},
		{
			name: "a empty b has values",
			a:    map[string]uint64{},
			b:    map[string]uint64{"X": 3},
			want: map[string]uint64{"X": 3},
		},
		{
			name: "a has values b empty",
			a:    map[string]uint64{"X": 3},
			b:    map[string]uint64{},
			want: map[string]uint64{"X": 3},
		},
		{
			name: "equal clocks produce same result",
			a:    map[string]uint64{"A": 2, "B": 2},
			b:    map[string]uint64{"A": 2, "B": 2},
			want: map[string]uint64{"A": 2, "B": 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeVersions(tt.a, tt.b)
			if len(got) != len(tt.want) {
				t.Fatalf("MergeVersions result has %d keys, want %d", len(got), len(tt.want))
			}
			for k, wantV := range tt.want {
				if got[k] != wantV {
					t.Errorf("MergeVersions[%q] = %d, want %d", k, got[k], wantV)
				}
			}
		})
	}
}

// TestMergeVersions_DoesNotMutateInputs verifies that the input maps are not
// modified by MergeVersions.
func TestMergeVersions_DoesNotMutateInputs(t *testing.T) {
	a := map[string]uint64{"A": 1}
	b := map[string]uint64{"B": 2}

	aCopy := map[string]uint64{"A": 1}
	bCopy := map[string]uint64{"B": 2}

	_ = MergeVersions(a, b)

	for k, v := range aCopy {
		if a[k] != v {
			t.Errorf("input a was mutated: key %q changed from %d to %d", k, v, a[k])
		}
	}
	if len(a) != len(aCopy) {
		t.Errorf("input a changed length: got %d, want %d", len(a), len(aCopy))
	}

	for k, v := range bCopy {
		if b[k] != v {
			t.Errorf("input b was mutated: key %q changed from %d to %d", k, v, b[k])
		}
	}
	if len(b) != len(bCopy) {
		t.Errorf("input b changed length: got %d, want %d", len(b), len(bCopy))
	}
}
