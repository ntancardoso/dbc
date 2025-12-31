package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ntancardoso/dbc/internal/models"
)

type SnapshotStorage struct {
	baseDir string
}

func NewSnapshotStorage(baseDir string) *SnapshotStorage {
	return &SnapshotStorage{
		baseDir: baseDir,
	}
}

func (s *SnapshotStorage) Save(snapshot *models.SchemaSnapshot) error {
	if err := os.MkdirAll(s.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	filename := fmt.Sprintf("%s_%s.json", snapshot.Key, snapshot.Timestamp.Format("20060102_150405"))
	filepath := filepath.Join(s.baseDir, filename)

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write snapshot: %w", err)
	}

	return nil
}

func (s *SnapshotStorage) Load(key string) (*models.SchemaSnapshot, error) {
	pattern := filepath.Join(s.baseDir, fmt.Sprintf("%s_*.json", key))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find snapshots: %w", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no snapshot found with key: %s", key)
	}

	sort.Strings(matches)
	latestFile := matches[len(matches)-1]

	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot: %w", err)
	}

	var snapshot models.SchemaSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	return &snapshot, nil
}

func (s *SnapshotStorage) List() ([]SnapshotInfo, error) {
	pattern := filepath.Join(s.baseDir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	snapshotMap := make(map[string]SnapshotInfo)
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}

		var snapshot models.SchemaSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue
		}

		if existing, ok := snapshotMap[snapshot.Key]; ok {
			if snapshot.Timestamp.After(existing.Timestamp) {
				snapshotMap[snapshot.Key] = SnapshotInfo{
					Key:       snapshot.Key,
					Database:  snapshot.Database,
					Timestamp: snapshot.Timestamp,
					Tables:    len(snapshot.Tables),
					FilePath:  match,
				}
			}
		} else {
			snapshotMap[snapshot.Key] = SnapshotInfo{
				Key:       snapshot.Key,
				Database:  snapshot.Database,
				Timestamp: snapshot.Timestamp,
				Tables:    len(snapshot.Tables),
				FilePath:  match,
			}
		}
	}

	var snapshots []SnapshotInfo
	for _, info := range snapshotMap {
		snapshots = append(snapshots, info)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Timestamp.After(snapshots[j].Timestamp)
	})

	return snapshots, nil
}

func (s *SnapshotStorage) Delete(key string) error {
	pattern := filepath.Join(s.baseDir, fmt.Sprintf("%s_*.json", key))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find snapshots: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no snapshot found with key: %s", key)
	}

	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			return fmt.Errorf("failed to delete snapshot: %w", err)
		}
	}

	return nil
}

type SnapshotInfo struct {
	Key       string
	Database  string
	Timestamp time.Time
	Tables    int
	FilePath  string
}
