package core

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ntancardoso/dbc/internal/models"
)

func CompareSnapshots(baseline, target *models.SchemaSnapshot) *models.ChangeSet {
	changeSet := &models.ChangeSet{
		Summary: models.ChangeSummary{},
	}

	baselineTables := make(map[string]models.Table)
	for _, table := range baseline.Tables {
		baselineTables[table.Name] = table
	}

	targetTables := make(map[string]models.Table)
	for _, table := range target.Tables {
		targetTables[table.Name] = table
	}

	for _, targetTable := range target.Tables {
		if baselineTable, exists := baselineTables[targetTable.Name]; exists {
			diff := compareTables(baselineTable, targetTable)
			if hasChanges(diff) {
				changeSet.TablesModified = append(changeSet.TablesModified, diff)
				changeSet.Summary.TablesModified++
			}
		} else {
			changeSet.TablesAdded = append(changeSet.TablesAdded, targetTable)
			changeSet.Summary.TablesAdded++
		}
	}

	for _, baselineTable := range baseline.Tables {
		if _, exists := targetTables[baselineTable.Name]; !exists {
			changeSet.TablesRemoved = append(changeSet.TablesRemoved, baselineTable)
			changeSet.Summary.TablesRemoved++
		}
	}

	return changeSet
}

func compareTables(baseline, target models.Table) models.TableDiff {
	diff := models.TableDiff{
		Name: baseline.Name,
	}

	// Compare columns
	baselineColumns := make(map[string]models.Column)
	for _, col := range baseline.Columns {
		baselineColumns[col.Name] = col
	}

	targetColumns := make(map[string]models.Column)
	for _, col := range target.Columns {
		targetColumns[col.Name] = col
	}

	for _, targetCol := range target.Columns {
		if baselineCol, exists := baselineColumns[targetCol.Name]; exists {
			if !columnsEqual(baselineCol, targetCol) {
				diff.ColumnsModified = append(diff.ColumnsModified, models.ColumnDiff{
					Name:   targetCol.Name,
					Before: baselineCol,
					After:  targetCol,
				})
			}
		} else {
			diff.ColumnsAdded = append(diff.ColumnsAdded, targetCol)
		}
	}

	for _, baselineCol := range baseline.Columns {
		if _, exists := targetColumns[baselineCol.Name]; !exists {
			diff.ColumnsRemoved = append(diff.ColumnsRemoved, baselineCol)
		}
	}

	// Compare indexes with modification detection
	baselineIndexes := make(map[string]models.Index)
	for _, idx := range baseline.Indexes {
		baselineIndexes[idx.Name] = idx
	}

	targetIndexes := make(map[string]models.Index)
	for _, idx := range target.Indexes {
		targetIndexes[idx.Name] = idx
	}

	for _, targetIdx := range target.Indexes {
		if baselineIdx, exists := baselineIndexes[targetIdx.Name]; exists {
			// Check if index was modified
			if !indexesEqual(baselineIdx, targetIdx) {
				diff.IndexesModified = append(diff.IndexesModified, models.IndexDiff{
					Name:   targetIdx.Name,
					Before: baselineIdx,
					After:  targetIdx,
				})
			}
		} else {
			diff.IndexesAdded = append(diff.IndexesAdded, targetIdx)
		}
	}

	for _, baselineIdx := range baseline.Indexes {
		if _, exists := targetIndexes[baselineIdx.Name]; !exists {
			diff.IndexesRemoved = append(diff.IndexesRemoved, baselineIdx)
		}
	}

	// Compare foreign keys with modification detection
	baselineFKs := make(map[string]models.ForeignKey)
	for _, fk := range baseline.ForeignKeys {
		baselineFKs[fk.Name] = fk
	}

	targetFKs := make(map[string]models.ForeignKey)
	for _, fk := range target.ForeignKeys {
		targetFKs[fk.Name] = fk
	}

	for _, targetFK := range target.ForeignKeys {
		if baselineFK, exists := baselineFKs[targetFK.Name]; exists {
			// Check if foreign key was modified
			if !foreignKeysEqual(baselineFK, targetFK) {
				diff.FKModified = append(diff.FKModified, models.ForeignKeyDiff{
					Name:   targetFK.Name,
					Before: baselineFK,
					After:  targetFK,
				})
			}
		} else {
			diff.FKAdded = append(diff.FKAdded, targetFK)
		}
	}

	for _, baselineFK := range baseline.ForeignKeys {
		if _, exists := targetFKs[baselineFK.Name]; !exists {
			diff.FKRemoved = append(diff.FKRemoved, baselineFK)
		}
	}

	// Compare row counts
	if baseline.RowCount != target.RowCount {
		change := target.RowCount - baseline.RowCount
		diff.RowCountChange = &change
	}

	// Compare checksums
	if baseline.Checksum != "" && target.Checksum != "" {
		if baseline.Checksum != target.Checksum {
			diff.ChecksumChanged = true
		}
	}

	return diff
}

func hasChanges(diff models.TableDiff) bool {
	return len(diff.ColumnsAdded) > 0 ||
		len(diff.ColumnsRemoved) > 0 ||
		len(diff.ColumnsModified) > 0 ||
		len(diff.IndexesAdded) > 0 ||
		len(diff.IndexesRemoved) > 0 ||
		len(diff.IndexesModified) > 0 ||
		len(diff.FKAdded) > 0 ||
		len(diff.FKRemoved) > 0 ||
		len(diff.FKModified) > 0 ||
		diff.RowCountChange != nil ||
		diff.ChecksumChanged
}

func columnsEqual(a, b models.Column) bool {
	return a.Name == b.Name &&
		a.ColumnType == b.ColumnType &&
		a.IsNullable == b.IsNullable &&
		a.Key == b.Key &&
		((a.DefaultValue == nil && b.DefaultValue == nil) ||
			(a.DefaultValue != nil && b.DefaultValue != nil && *a.DefaultValue == *b.DefaultValue))
}

func indexesEqual(a, b models.Index) bool {
	if a.Name != b.Name ||
		a.IsUnique != b.IsUnique ||
		a.IsPrimary != b.IsPrimary ||
		a.Type != b.Type {
		return false
	}
	// Use reflect.DeepEqual to compare column slices
	return reflect.DeepEqual(a.Columns, b.Columns)
}

func foreignKeysEqual(a, b models.ForeignKey) bool {
	return a.Name == b.Name &&
		a.Column == b.Column &&
		a.ReferencedTable == b.ReferencedTable &&
		a.ReferencedColumn == b.ReferencedColumn &&
		a.OnDelete == b.OnDelete &&
		a.OnUpdate == b.OnUpdate
}

func FormatChangeSet(changeSet *models.ChangeSet, baselineKey, targetKey string) string {
	output := fmt.Sprintf("=== Schema Comparison: %s → %s ===\n\n", baselineKey, targetKey)

	output += "Summary:\n"
	output += fmt.Sprintf("  Tables Added:    %d\n", changeSet.Summary.TablesAdded)
	output += fmt.Sprintf("  Tables Removed:  %d\n", changeSet.Summary.TablesRemoved)
	output += fmt.Sprintf("  Tables Modified: %d\n", changeSet.Summary.TablesModified)
	output += "\n"

	if len(changeSet.TablesAdded) > 0 {
		output += "Added Tables:\n"
		for _, table := range changeSet.TablesAdded {
			output += fmt.Sprintf("  + %s (%d columns, %d rows)\n", table.Name, len(table.Columns), table.RowCount)
		}
		output += "\n"
	}

	if len(changeSet.TablesRemoved) > 0 {
		output += "Removed Tables:\n"
		for _, table := range changeSet.TablesRemoved {
			output += fmt.Sprintf("  - %s (%d columns, %d rows)\n", table.Name, len(table.Columns), table.RowCount)
		}
		output += "\n"
	}

	if len(changeSet.TablesModified) > 0 {
		output += "Modified Tables:\n"
		for _, diff := range changeSet.TablesModified {
			output += fmt.Sprintf("  ~ %s\n", diff.Name)

			if len(diff.ColumnsAdded) > 0 {
				output += "    Added Columns:\n"
				for _, col := range diff.ColumnsAdded {
					output += fmt.Sprintf("      + %s (%s)\n", col.Name, col.ColumnType)
				}
			}

			if len(diff.ColumnsRemoved) > 0 {
				output += "    Removed Columns:\n"
				for _, col := range diff.ColumnsRemoved {
					output += fmt.Sprintf("      - %s (%s)\n", col.Name, col.ColumnType)
				}
			}

			if len(diff.ColumnsModified) > 0 {
				output += "    Modified Columns:\n"
				for _, colDiff := range diff.ColumnsModified {
					output += fmt.Sprintf("      ~ %s: %s → %s\n", colDiff.Name, colDiff.Before.ColumnType, colDiff.After.ColumnType)
				}
			}

			if len(diff.IndexesAdded) > 0 {
				output += "    Added Indexes:\n"
				for _, idx := range diff.IndexesAdded {
					output += fmt.Sprintf("      + %s\n", idx.Name)
				}
			}

			if len(diff.IndexesRemoved) > 0 {
				output += "    Removed Indexes:\n"
				for _, idx := range diff.IndexesRemoved {
					output += fmt.Sprintf("      - %s\n", idx.Name)
				}
			}

			if len(diff.IndexesModified) > 0 {
				output += "    Modified Indexes:\n"
				for _, idxDiff := range diff.IndexesModified {
					output += fmt.Sprintf("      ~ %s: unique=%v→%v, primary=%v→%v\n",
						idxDiff.Name,
						idxDiff.Before.IsUnique, idxDiff.After.IsUnique,
						idxDiff.Before.IsPrimary, idxDiff.After.IsPrimary)
				}
			}

			if len(diff.FKAdded) > 0 {
				output += "    Added Foreign Keys:\n"
				for _, fk := range diff.FKAdded {
					output += fmt.Sprintf("      + %s → %s(%s)\n", fk.Column, fk.ReferencedTable, fk.ReferencedColumn)
				}
			}

			if len(diff.FKRemoved) > 0 {
				output += "    Removed Foreign Keys:\n"
				for _, fk := range diff.FKRemoved {
					output += fmt.Sprintf("      - %s → %s(%s)\n", fk.Column, fk.ReferencedTable, fk.ReferencedColumn)
				}
			}

			if len(diff.FKModified) > 0 {
				output += "    Modified Foreign Keys:\n"
				for _, fkDiff := range diff.FKModified {
					output += fmt.Sprintf("      ~ %s: %s(%s)→%s(%s), OnDelete:%s→%s\n",
						fkDiff.Name,
						fkDiff.Before.ReferencedTable, fkDiff.Before.ReferencedColumn,
						fkDiff.After.ReferencedTable, fkDiff.After.ReferencedColumn,
						fkDiff.Before.OnDelete, fkDiff.After.OnDelete)
				}
			}

			if diff.RowCountChange != nil && *diff.RowCountChange != 0 {
				sign := "+"
				if *diff.RowCountChange < 0 {
					sign = ""
				}
				output += fmt.Sprintf("    Row Count: %s%d\n", sign, *diff.RowCountChange)
			}

			if diff.ChecksumChanged {
				output += "    ⚠ Data Checksum Changed (data modified)\n"
			}

			output += "\n"
		}
	}

	if changeSet.Summary.TablesAdded == 0 && changeSet.Summary.TablesRemoved == 0 && changeSet.Summary.TablesModified == 0 {
		output += "No changes detected.\n"
	}

	return output
}

func FormatChangeSetJSON(changeSet *models.ChangeSet, baselineKey, targetKey string) (string, error) {
	report := map[string]interface{}{
		"baseline_key": baselineKey,
		"target_key":   targetKey,
		"summary":      changeSet.Summary,
		"changes": map[string]interface{}{
			"tables_added":    changeSet.TablesAdded,
			"tables_removed":  changeSet.TablesRemoved,
			"tables_modified": changeSet.TablesModified,
		},
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(data), nil
}
