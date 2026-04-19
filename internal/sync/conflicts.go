package sync

// VersionOrder describes the causal relationship between two vector clocks.
type VersionOrder int

const (
	// LocalNewer means the local vector clock strictly dominates the remote one:
	// every component in local is >= remote, and at least one is strictly greater.
	LocalNewer VersionOrder = iota
	// RemoteNewer means the remote vector clock strictly dominates the local one.
	RemoteNewer
	// Concurrent means neither clock dominates the other; a conflict exists.
	Concurrent
	// Equal means both clocks are identical across all peers.
	Equal
)

// CompareVersions determines the causal relationship between a local and a
// remote vector clock. Both maps are keyed by peer ID; absent entries are
// treated as zero.
func CompareVersions(local, remote map[string]uint64) VersionOrder {
	localGt := false
	remoteGt := false

	allKeys := map[string]bool{}
	for k := range local {
		allKeys[k] = true
	}
	for k := range remote {
		allKeys[k] = true
	}

	for k := range allKeys {
		lv := local[k]
		rv := remote[k]
		if lv > rv {
			localGt = true
		}
		if rv > lv {
			remoteGt = true
		}
	}

	switch {
	case localGt && remoteGt:
		return Concurrent
	case localGt:
		return LocalNewer
	case remoteGt:
		return RemoteNewer
	default:
		return Equal
	}
}

// MergeVersions returns a new vector clock whose components are the
// element-wise maximum of clocks a and b. This is the standard join operation
// for vector clocks and is used when accepting a remote change.
func MergeVersions(a, b map[string]uint64) map[string]uint64 {
	merged := make(map[string]uint64)
	for k, v := range a {
		merged[k] = v
	}
	for k, v := range b {
		if v > merged[k] {
			merged[k] = v
		}
	}
	return merged
}
