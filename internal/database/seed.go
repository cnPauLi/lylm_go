package database

import (
	_ "embed"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"lylme_spage/internal/logger"
	"lylme_spage/internal/models"
)

// seed.sql 为原版 install_struct.sql 的副本，用于首次初始化种子数据
//
//go:embed seed.sql
var seedSQL string

type insertStmt struct {
	table string
	cols  []string
	rows  [][]any
}

// Seed 当配置表为空时，从原版 SQL 导入种子数据
func Seed(db *gorm.DB, log *logger.Logger) error {
	var count int64
	if err := db.Model(&models.Config{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	stmts, err := parseInserts(seedSQL)
	if err != nil {
		return err
	}

	tx := db.Begin()
	for _, st := range stmts {
		if len(st.rows) == 0 || len(st.cols) == 0 {
			continue
		}
		colList := "`" + strings.Join(st.cols, "`,`") + "`"
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(st.cols)), ",")
		sql := "INSERT INTO `" + st.table + "` (" + colList + ") VALUES (" + placeholders + ")"
		for _, row := range st.rows {
			if err := tx.Exec(sql, row...).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	log.Infof("数据库初始化完成，已导入 %d 张表的种子数据", len(stmts))
	return nil
}

// parseInserts 从 SQL dump 中解析所有 INSERT 语句
func parseInserts(sqlText string) ([]insertStmt, error) {
	var stmts []insertStmt
	for _, raw := range splitStatements(sqlText) {
		if st, ok := parseInsert(raw); ok {
			stmts = append(stmts, st)
		}
	}
	return stmts, nil
}

// splitStatements 按分号切分语句，忽略引号内的分号
func splitStatements(s string) []string {
	var out []string
	var b strings.Builder
	inQuote := false
	escaped := false
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			b.WriteByte(c)
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
			} else if c == quote {
				inQuote = false
			}
			continue
		}
		switch c {
		case '\'', '"':
			inQuote = true
			quote = c
			b.WriteByte(c)
		case ';':
			out = append(out, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	if strings.TrimSpace(b.String()) != "" {
		out = append(out, b.String())
	}
	return out
}

// parseInsert 解析单条 INSERT 语句
func parseInsert(stmt string) (insertStmt, bool) {
	s := strings.TrimSpace(stmt)
	if !strings.HasPrefix(strings.ToUpper(s), "INSERT INTO") {
		return insertStmt{}, false
	}
	s = strings.TrimSpace(s[len("INSERT INTO"):])

	var table string
	if len(s) > 0 && s[0] == '`' {
		end := strings.IndexByte(s[1:], '`')
		if end < 0 {
			return insertStmt{}, false
		}
		table = s[1 : 1+end]
		s = s[1+end+1:]
	} else {
		idx := strings.IndexByte(s, '(')
		if idx < 0 {
			return insertStmt{}, false
		}
		table = strings.TrimSpace(s[:idx])
		s = s[idx:]
	}

	open := strings.IndexByte(s, '(')
	if open < 0 {
		return insertStmt{}, false
	}
	close := matchParen(s, open)
	if close < 0 {
		return insertStmt{}, false
	}
	cols := splitColumns(s[open+1 : close])

	rest := s[close+1:]
	vi := indexFold(rest, "VALUES")
	if vi < 0 {
		return insertStmt{}, false
	}
	rows := parseValueTuples(rest[vi+len("VALUES"):])

	return insertStmt{table: table, cols: cols, rows: rows}, true
}

func matchParen(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func splitColumns(s string) []string {
	parts := strings.Split(s, ",")
	cols := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "`")
		if p != "" {
			cols = append(cols, p)
		}
	}
	return cols
}

// parseValueTuples 解析 VALUES 后的多个元组
func parseValueTuples(s string) [][]any {
	var rows [][]any
	i := 0
	n := len(s)
	for i < n {
		for i < n && s[i] != '(' {
			i++
		}
		if i >= n {
			break
		}
		i++ // 跳过 '('
		var row []any
		for i < n {
			for i < n && (s[i] == ' ' || s[i] == ',' || s[i] == '\n' || s[i] == '\r' || s[i] == '\t') {
				i++
			}
			if i >= n {
				break
			}
			if s[i] == ')' {
				i++
				break
			}
			if s[i] == '\'' {
				i++
				var sb strings.Builder
				for i < n {
					c := s[i]
					if c == '\\' && i+1 < n {
						sb.WriteByte(unescapeMySQL(s[i+1]))
						i += 2
						continue
					}
					if c == '\'' {
						if i+1 < n && s[i+1] == '\'' {
							sb.WriteByte('\'')
							i += 2
							continue
						}
						i++
						break
					}
					sb.WriteByte(c)
					i++
				}
				row = append(row, sb.String())
			} else {
				start := i
				for i < n && s[i] != ',' && s[i] != ')' {
					i++
				}
				tok := strings.TrimSpace(s[start:i])
				switch {
				case strings.EqualFold(tok, "NULL"):
					row = append(row, nil)
				default:
					if iv, err := strconv.ParseInt(tok, 10, 64); err == nil {
						row = append(row, iv)
					} else if fv, err := strconv.ParseFloat(tok, 64); err == nil {
						row = append(row, fv)
					} else {
						row = append(row, tok)
					}
				}
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// unescapeMySQL 将 MySQL 字符串转义还原为原始字符
func unescapeMySQL(c byte) byte {
	switch c {
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case '0':
		return 0
	case 'b':
		return '\b'
	case 'Z':
		return 0x1a
	case '\\', '\'', '"', '%', '_':
		return c
	default:
		return c
	}
}

func indexFold(s, sub string) int {
	return strings.Index(strings.ToUpper(s), strings.ToUpper(sub))
}
