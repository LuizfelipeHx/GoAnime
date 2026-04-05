package main

import "fmt"

// GetPlayQueue returns the current playback queue.
func (a *App) GetPlayQueue() []QueueEntry {
	a.playQueueMu.RLock()
	defer a.playQueueMu.RUnlock()

	out := make([]QueueEntry, len(a.playQueue))
	copy(out, a.playQueue)
	return out
}

// AddToQueue appends an entry to the playback queue.
func (a *App) AddToQueue(entry QueueEntry) error {
	a.playQueueMu.Lock()
	defer a.playQueueMu.Unlock()

	a.playQueue = append(a.playQueue, entry)
	return nil
}

// RemoveFromQueue removes an entry at the given index.
func (a *App) RemoveFromQueue(index int) error {
	a.playQueueMu.Lock()
	defer a.playQueueMu.Unlock()

	if index < 0 || index >= len(a.playQueue) {
		return fmt.Errorf("indice %d fora do intervalo (0-%d)", index, len(a.playQueue)-1)
	}

	a.playQueue = append(a.playQueue[:index], a.playQueue[index+1:]...)
	return nil
}

// ClearQueue removes all entries from the playback queue.
func (a *App) ClearQueue() error {
	a.playQueueMu.Lock()
	defer a.playQueueMu.Unlock()

	a.playQueue = nil
	return nil
}

// ReorderQueue moves an entry from one index to another.
func (a *App) ReorderQueue(fromIndex int, toIndex int) error {
	a.playQueueMu.Lock()
	defer a.playQueueMu.Unlock()

	n := len(a.playQueue)
	if fromIndex < 0 || fromIndex >= n {
		return fmt.Errorf("indice origem %d fora do intervalo (0-%d)", fromIndex, n-1)
	}
	if toIndex < 0 || toIndex >= n {
		return fmt.Errorf("indice destino %d fora do intervalo (0-%d)", toIndex, n-1)
	}
	if fromIndex == toIndex {
		return nil
	}

	entry := a.playQueue[fromIndex]
	// Remove from source position
	a.playQueue = append(a.playQueue[:fromIndex], a.playQueue[fromIndex+1:]...)
	// Insert at destination position
	rear := make([]QueueEntry, len(a.playQueue[toIndex:]))
	copy(rear, a.playQueue[toIndex:])
	a.playQueue = append(a.playQueue[:toIndex], entry)
	a.playQueue = append(a.playQueue, rear...)
	return nil
}
