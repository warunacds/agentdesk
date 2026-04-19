package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// Collection represents a named group of skill file paths.
type Collection struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	SkillPaths []string `json:"skillPaths"`
	Created    int64    `json:"created"`
	Modified   int64    `json:"modified"`
}

// CollectionsStore is the top-level JSON structure persisted to disk.
type CollectionsStore struct {
	Collections []Collection `json:"collections"`
}

// collectionsFilePath returns ~/.agentdesk/collections.json.
func collectionsFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentdesk", "collections.json")
}

// LoadCollections reads the collections store from disk.
// Returns an empty store if the file does not exist or is unreadable.
func LoadCollections() CollectionsStore {
	data, err := os.ReadFile(collectionsFilePath())
	if err != nil {
		return CollectionsStore{Collections: []Collection{}}
	}
	var store CollectionsStore
	if err := json.Unmarshal(data, &store); err != nil {
		return CollectionsStore{Collections: []Collection{}}
	}
	if store.Collections == nil {
		store.Collections = []Collection{}
	}
	return store
}

// SaveCollections writes the collections store to disk, creating
// the ~/.agentdesk/ directory if it does not exist.
func SaveCollections(store CollectionsStore) error {
	fp := collectionsFilePath()
	if err := os.MkdirAll(filepath.Dir(fp), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fp, data, 0644)
}

// FindCollectionIndex returns the index of the collection with the given ID,
// or -1 if not found.
func FindCollectionIndex(store *CollectionsStore, id string) int {
	for i, c := range store.Collections {
		if c.ID == id {
			return i
		}
	}
	return -1
}

// NewCollection creates a Collection with a new UUID and timestamps.
func NewCollection(name string) Collection {
	now := time.Now().Unix()
	return Collection{
		ID:         uuid.New().String(),
		Name:       name,
		SkillPaths: []string{},
		Created:    now,
		Modified:   now,
	}
}

// AddPath appends a skill path to a collection if not already present.
// Returns true if the path was added.
func (c *Collection) AddPath(path string) bool {
	for _, p := range c.SkillPaths {
		if p == path {
			return false
		}
	}
	c.SkillPaths = append(c.SkillPaths, path)
	c.Modified = time.Now().Unix()
	return true
}

// RemovePath removes a skill path from a collection.
// Returns true if the path was found and removed.
func (c *Collection) RemovePath(path string) bool {
	for i, p := range c.SkillPaths {
		if p == path {
			c.SkillPaths = append(c.SkillPaths[:i], c.SkillPaths[i+1:]...)
			c.Modified = time.Now().Unix()
			return true
		}
	}
	return false
}
