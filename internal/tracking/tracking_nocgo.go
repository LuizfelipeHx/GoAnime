//go:build !cgo

package tracking

// newJsonTracker creates a LocalTracker backed by a JSON file.
// Used when CGO/SQLite is not available.
func newJsonTracker(dbPath string) *LocalTracker {
	trackerMutex.Lock()
	defer trackerMutex.Unlock()

	// Return cached tracker if the path matches
	if globalTracker != nil && globalTrackerPath == dbPath {
		return globalTracker
	}

	store := newJsonStorage(dbPath)
	tracker := &LocalTracker{jsonStore: store}

	globalTracker = tracker
	globalTrackerPath = dbPath
	return tracker
}

// init overrides NewLocalTracker to use JSON storage when CGO is disabled.
func init() {
	NewLocalTracker = func(dbPath string) *LocalTracker {
		return newJsonTracker(dbPath)
	}
	IsCgoEnabled = false
}
