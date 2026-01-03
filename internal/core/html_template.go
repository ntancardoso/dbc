package core

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Database Schema Comparison: {{.BaselineKey}} → {{.TargetKey}}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; line-height: 1.6; color: #333; background: #f5f5f5; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; border-radius: 8px 8px 0 0; }
        .header h1 { font-size: 28px; margin-bottom: 10px; }
        .header .comparison { font-size: 18px; opacity: 0.9; }
        .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; padding: 30px; background: #f8f9fa; }
        .summary-card { background: white; padding: 20px; border-radius: 6px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); text-align: center; }
        .summary-card .number { font-size: 36px; font-weight: bold; margin-bottom: 5px; }
        .summary-card .label { color: #666; font-size: 14px; text-transform: uppercase; letter-spacing: 0.5px; }
        .added .number { color: #10b981; }
        .removed .number { color: #ef4444; }
        .modified .number { color: #f59e0b; }
        .content { padding: 30px; }
        .section { margin-bottom: 30px; }
        .section h2 { font-size: 20px; margin-bottom: 15px; padding-bottom: 10px; border-bottom: 2px solid #e5e7eb; }
        .table-item { background: #f9fafb; padding: 15px; margin-bottom: 10px; border-radius: 6px; border-left: 4px solid #ddd; }
        .table-item.added { border-left-color: #10b981; background: #ecfdf5; }
        .table-item.removed { border-left-color: #ef4444; background: #fef2f2; }
        .table-item.modified { border-left-color: #f59e0b; background: #fffbeb; }
        .table-name { font-weight: 600; font-size: 16px; margin-bottom: 8px; }
        .table-meta { font-size: 14px; color: #666; }
        .change-list { margin-top: 10px; padding-left: 20px; }
        .change-item { padding: 4px 0; font-size: 14px; }
        .change-item.add { color: #10b981; }
        .change-item.remove { color: #ef4444; }
        .change-item.modify { color: #f59e0b; }
        .change-item.warning { color: #dc2626; font-weight: 500; }
        .icon { margin-right: 8px; }
        .no-changes { text-align: center; padding: 60px 20px; color: #9ca3af; }
        .no-changes .icon { font-size: 48px; margin-bottom: 15px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Database Schema Comparison</h1>
            <div class="comparison">{{.BaselineKey}} → {{.TargetKey}}</div>
        </div>

        <div class="summary">
            <div class="summary-card added">
                <div class="number">{{.Summary.TablesAdded}}</div>
                <div class="label">Tables Added</div>
            </div>
            <div class="summary-card removed">
                <div class="number">{{.Summary.TablesRemoved}}</div>
                <div class="label">Tables Removed</div>
            </div>
            <div class="summary-card modified">
                <div class="number">{{.Summary.TablesModified}}</div>
                <div class="label">Tables Modified</div>
            </div>
        </div>

        <div class="content">
            {{if .TablesAdded}}
            <div class="section">
                <h2>Added Tables</h2>
                {{range .TablesAdded}}
                <div class="table-item added">
                    <div class="table-name">+ {{.Name}}</div>
                    <div class="table-meta">{{len .Columns}} columns, {{.RowCount}} rows</div>
                </div>
                {{end}}
            </div>
            {{end}}

            {{if .TablesRemoved}}
            <div class="section">
                <h2>Removed Tables</h2>
                {{range .TablesRemoved}}
                <div class="table-item removed">
                    <div class="table-name">- {{.Name}}</div>
                    <div class="table-meta">{{len .Columns}} columns, {{.RowCount}} rows</div>
                </div>
                {{end}}
            </div>
            {{end}}

            {{if .TablesModified}}
            <div class="section">
                <h2>Modified Tables</h2>
                {{range .TablesModified}}
                <div class="table-item modified">
                    <div class="table-name">~ {{.Name}}</div>
                    <div class="change-list">
                        {{range .ColumnsAdded}}
                        <div class="change-item add"><span class="icon">+</span>Column: {{.Name}} ({{.ColumnType}})</div>
                        {{end}}
                        {{range .ColumnsRemoved}}
                        <div class="change-item remove"><span class="icon">-</span>Column: {{.Name}} ({{.ColumnType}})</div>
                        {{end}}
                        {{range .ColumnsModified}}
                        <div class="change-item modify"><span class="icon">~</span>Column: {{.Name}} ({{.Before.ColumnType}} → {{.After.ColumnType}})</div>
                        {{end}}
                        {{range .IndexesAdded}}
                        <div class="change-item add"><span class="icon">+</span>Index: {{.Name}}</div>
                        {{end}}
                        {{range .IndexesRemoved}}
                        <div class="change-item remove"><span class="icon">-</span>Index: {{.Name}}</div>
                        {{end}}
                        {{range .FKAdded}}
                        <div class="change-item add"><span class="icon">+</span>Foreign Key: {{.Name}}</div>
                        {{end}}
                        {{range .FKRemoved}}
                        <div class="change-item remove"><span class="icon">-</span>Foreign Key: {{.Name}}</div>
                        {{end}}
                        {{if .RowCountChange}}
                        <div class="change-item modify"><span class="icon">~</span>Row Count: {{if gt (deref .RowCountChange) 0}}+{{end}}{{deref .RowCountChange}}</div>
                        {{end}}
                        {{if .ChecksumChanged}}
                        <div class="change-item warning"><span class="icon">⚠</span>Data Checksum Changed (data modified)</div>
                        {{end}}
                    </div>
                </div>
                {{end}}
            </div>
            {{end}}

            {{if and (eq .Summary.TablesAdded 0) (eq .Summary.TablesRemoved 0) (eq .Summary.TablesModified 0)}}
            <div class="no-changes">
                <div class="icon">✓</div>
                <div>No changes detected</div>
            </div>
            {{end}}
        </div>
    </div>
</body>
</html>`
