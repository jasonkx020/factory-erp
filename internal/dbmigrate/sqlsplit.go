package dbmigrate

import "strings"

// SplitStatements splits SQL into executable statements, respecting $$ blocks and line comments.
func SplitStatements(sql string) []string {
	var out []string
	var b strings.Builder
	inSingle := false
	inDouble := false
	inDollar := false
	dollarTag := ""
	i := 0
	for i < len(sql) {
		ch := sql[i]
		if !inSingle && !inDouble && !inDollar && i+1 < len(sql) && ch == '-' && sql[i+1] == '-' {
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			continue
		}
		if !inDouble && !inDollar && ch == '\'' {
			if inSingle && i+1 < len(sql) && sql[i+1] == '\'' {
				b.WriteByte(ch)
				b.WriteByte(sql[i+1])
				i += 2
				continue
			}
			inSingle = !inSingle
			b.WriteByte(ch)
			i++
			continue
		}
		if !inSingle && !inDollar && ch == '"' {
			inDouble = !inDouble
			b.WriteByte(ch)
			i++
			continue
		}
		if !inSingle && !inDouble && ch == '$' {
			if !inDollar {
				j := i + 1
				for j < len(sql) && (sql[j] == '_' || (sql[j] >= 'a' && sql[j] <= 'z') || (sql[j] >= 'A' && sql[j] <= 'Z') || (sql[j] >= '0' && sql[j] <= '9')) {
					j++
				}
				if j < len(sql) && sql[j] == '$' {
					dollarTag = sql[i : j+1]
					inDollar = true
					b.WriteString(dollarTag)
					i = j + 1
					continue
				}
			} else if strings.HasPrefix(sql[i:], dollarTag) {
				b.WriteString(dollarTag)
				i += len(dollarTag)
				inDollar = false
				dollarTag = ""
				continue
			}
		}
		if !inSingle && !inDouble && !inDollar && ch == ';' {
			if stmt := strings.TrimSpace(b.String()); stmt != "" {
				out = append(out, stmt)
			}
			b.Reset()
			i++
			continue
		}
		b.WriteByte(ch)
		i++
	}
	if stmt := strings.TrimSpace(b.String()); stmt != "" {
		out = append(out, stmt)
	}
	return out
}
