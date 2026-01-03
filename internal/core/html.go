package core

import (
	"bytes"
	"html/template"

	"github.com/ntancardoso/dbc/internal/models"
)

type TableDiffView struct {
	Name            string
	ColumnsAdded    []models.Column
	ColumnsRemoved  []models.Column
	ColumnsModified []models.ColumnDiff
	IndexesAdded    []models.Index
	IndexesRemoved  []models.Index
	FKAdded         []models.ForeignKey
	FKRemoved       []models.ForeignKey
	RowCountChange  *int64
	ChecksumChanged bool
}

func FormatChangeSetHTML(changeSet *models.ChangeSet, baselineKey, targetKey string) (string, error) {
	funcMap := template.FuncMap{
		"deref": func(p *int64) int64 {
			if p == nil {
				return 0
			}
			return *p
		},
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	modifiedViews := make([]TableDiffView, len(changeSet.TablesModified))
	for i, diff := range changeSet.TablesModified {
		modifiedViews[i] = TableDiffView{
			Name:            diff.Name,
			ColumnsAdded:    diff.ColumnsAdded,
			ColumnsRemoved:  diff.ColumnsRemoved,
			ColumnsModified: diff.ColumnsModified,
			IndexesAdded:    diff.IndexesAdded,
			IndexesRemoved:  diff.IndexesRemoved,
			FKAdded:         diff.FKAdded,
			FKRemoved:       diff.FKRemoved,
			RowCountChange:  diff.RowCountChange,
			ChecksumChanged: diff.ChecksumChanged,
		}
	}

	data := struct {
		BaselineKey    string
		TargetKey      string
		Summary        models.ChangeSummary
		TablesAdded    []models.Table
		TablesRemoved  []models.Table
		TablesModified []TableDiffView
	}{
		BaselineKey:    baselineKey,
		TargetKey:      targetKey,
		Summary:        changeSet.Summary,
		TablesAdded:    changeSet.TablesAdded,
		TablesRemoved:  changeSet.TablesRemoved,
		TablesModified: modifiedViews,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
