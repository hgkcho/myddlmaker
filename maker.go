package myddlmaker

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"os"
	"strings"
)

type Config struct {
	DB            *DBConfig
	OutFilePath   string
	OutGoFilePath string
	PackageName   string
}

type DBConfig struct {
	Driver  string
	Engine  string
	Charset string
}

type Maker struct {
	config  *Config
	structs []any
	tables  []*table
}

func New(config *Config) (*Maker, error) {
	return &Maker{
		config: config,
	}, nil
}

func (m *Maker) AddStructs(structs ...any) {
	m.structs = append(m.structs, structs...)
}

// GenerateFile opens
func (m *Maker) GenerateFile() error {
	f, err := os.Create(m.config.OutFilePath)
	if err != nil {
		return fmt.Errorf("myddlmaker: failed to open %q: %w", m.config.OutFilePath, err)
	}
	defer f.Close()

	if err := m.Generate(f); err != nil {
		return fmt.Errorf("myddlmaker: failed to generate ddl: %w", err)
	}

	return f.Close()
}

func (m *Maker) Generate(w io.Writer) error {
	var buf bytes.Buffer
	if err := m.parse(); err != nil {
		return err
	}

	buf.WriteString("SET foreign_key_checks=0;\n")
	for _, table := range m.tables {
		m.generateTable(&buf, table)
	}

	buf.WriteString("SET foreign_key_checks=1;\n")

	if _, err := buf.WriteTo(w); err != nil {
		return err
	}
	return nil
}

func (m *Maker) parse() error {
	m.tables = make([]*table, len(m.structs))
	for i, s := range m.structs {
		tbl, err := newTable(s)
		if err != nil {
			return fmt.Errorf("myddlmaker: failed to parse: %w", err)
		}
		m.tables[i] = tbl
	}
	return nil
}

func (m *Maker) generateTable(w io.Writer, table *table) {
	fmt.Fprintf(w, "DROP TABLE IF EXISTS %s;\n\n", quote(table.name))
	fmt.Fprintf(w, "CREATE TABLE %s (\n", quote(table.name))
	for _, col := range table.columns {
		m.generateColumn(w, col)
	}
	m.generateIndex(w, table)
	fmt.Fprintf(w, "    PRIMARY KEY (%s)\n", strings.Join(quoteAll(table.primaryKey.columns), ", "))

	fmt.Fprintf(w, ")")
	if m.config != nil && m.config.DB != nil {
		if engine := m.config.DB.Engine; engine != "" {
			fmt.Fprintf(w, " ENGINE = %s", engine)
		}
		if charset := m.config.DB.Charset; charset != "" {
			fmt.Fprintf(w, " DEFAULT CHARACTER SET = %s", charset)
		}
	}
	fmt.Fprintf(w, ";\n\n")
}

func (m *Maker) generateColumn(w io.Writer, col *column) {
	io.WriteString(w, "    ")
	io.WriteString(w, quote(col.name))
	io.WriteString(w, " ")
	io.WriteString(w, col.typ)
	if col.size != 0 {
		fmt.Fprintf(w, "(%d)", col.size)
	}
	if col.unsigned {
		io.WriteString(w, " unsigned")
	}
	if col.null {
		io.WriteString(w, " NULL")
	} else {
		io.WriteString(w, " NOT NULL")
	}
	if col.def != "" {
		io.WriteString(w, " DEFAULT ")
		io.WriteString(w, col.def)
	}
	if col.autoIncr {
		io.WriteString(w, " AUTO_INCREMENT")
	}
	io.WriteString(w, ",\n")
}

func (m *Maker) generateIndex(w io.Writer, table *table) {
	for _, idx := range table.indexes {
		io.WriteString(w, "    INDEX ")
		io.WriteString(w, quote(idx.name))
		io.WriteString(w, " (")
		io.WriteString(w, strings.Join(quoteAll(idx.columns), ", "))
		io.WriteString(w, ")")
		if idx.invisible {
			io.WriteString(w, " INVISIBLE")
		}
		if idx.comment != "" {
			io.WriteString(w, " COMMENT ")
			io.WriteString(w, stringQuote(idx.comment))
		}
		io.WriteString(w, ",\n")
	}

	for _, idx := range table.uniqueIndexes {
		io.WriteString(w, "    UNIQUE ")
		io.WriteString(w, quote(idx.name))
		io.WriteString(w, " (")
		io.WriteString(w, strings.Join(quoteAll(idx.columns), ", "))
		io.WriteString(w, ")")
		if idx.invisible {
			io.WriteString(w, " INVISIBLE")
		}
		if idx.comment != "" {
			io.WriteString(w, " COMMENT ")
			io.WriteString(w, stringQuote(idx.comment))
		}
		io.WriteString(w, ",\n")
	}

	for _, idx := range table.foreignKeys {
		io.WriteString(w, "    CONSTRAINT ")
		io.WriteString(w, quote(idx.name))
		io.WriteString(w, " FOREIGN KEY (")
		io.WriteString(w, strings.Join(quoteAll(idx.columns), ", "))
		io.WriteString(w, ") REFERENCES ")
		io.WriteString(w, quote(idx.table))
		io.WriteString(w, " (")
		io.WriteString(w, strings.Join(quoteAll(idx.references), ", "))
		io.WriteString(w, ")")
		if idx.onUpdate != "" {
			io.WriteString(w, " ON UPDATE ")
			io.WriteString(w, string(idx.onUpdate))
		}
		if idx.onDelete != "" {
			io.WriteString(w, " ON DELETE ")
			io.WriteString(w, string(idx.onDelete))
		}
		io.WriteString(w, ",\n")
	}
}

// quote quotes s with `s`.
func quote(s string) string {
	var buf strings.Builder
	// Strictly speaking, we need to count the number of back quotes in s.
	// However, in many cases, s doesn't include back quotes.
	buf.Grow(len(s) + len("``"))

	buf.WriteByte('`')
	for _, r := range s {
		if r == '`' {
			buf.WriteByte('`')
		}
		buf.WriteRune(r)
	}
	buf.WriteByte('`')
	return buf.String()
}

func quoteAll(strings []string) []string {
	ret := make([]string, len(strings))
	for i, s := range strings {
		ret[i] = quote(s)
	}
	return ret
}

// escape sequence table
// https://dev.mysql.com/doc/refman/8.0/en/string-literals.html
var stringQuoter = strings.NewReplacer(
	"\x00", `\0`,
	"'", `\'`,
	`"`, `\"`,
	"\b", `\b`,
	"\n", `\n`,
	"\r", `\r`,
	"\t", `\t`,
	"\x1a", `\Z`,
	"\\", `\\`,
)

// stringQuote quotes s with 's'.
func stringQuote(s string) string {
	var buf strings.Builder
	// Strictly speaking, we need to count the number of quotes in s.
	// However, in many cases, s doesn't include quotes.
	buf.Grow(len(s) + len("''"))

	buf.WriteByte('\'')
	stringQuoter.WriteString(&buf, s)
	buf.WriteByte('\'')
	return buf.String()
}

type PrimaryKey struct {
	columns []string
}

type primaryKey interface {
	PrimaryKey() *PrimaryKey
}

func NewPrimaryKey(field ...string) *PrimaryKey {
	return &PrimaryKey{
		columns: field,
	}
}

func (m *Maker) GenerateGoFile() error {
	f, err := os.Create(m.config.OutGoFilePath)
	if err != nil {
		return fmt.Errorf("myddlmaker: failed to open %q: %w", m.config.OutGoFilePath, err)
	}
	defer f.Close()

	if err := m.GenerateGo(f); err != nil {
		return fmt.Errorf("myddlmaker: failed to generate go file: %w", err)
	}

	return f.Close()
}

func (m *Maker) GenerateGo(w io.Writer) error {
	var buf bytes.Buffer
	if err := m.parse(); err != nil {
		return err
	}

	m.generateGoHeader(&buf)
	for _, table := range m.tables {
		m.generateGoTable(&buf, table)
	}

	source, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	_, err = w.Write(source)
	return err
}

func (m *Maker) generateGoHeader(w io.Writer) {
	io.WriteString(w, "// Code generated by https://github.com/shogo82148/myddlmaker; DO NOT EDIT.\n\n")
	fmt.Fprintf(w, "package %s\n\n", m.config.PackageName)
	fmt.Fprintf(w, `import (
		"context"
		"database/sql"
	)

	type execer interface {
		ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	}

	type queryer interface {
		QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
		QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	}

	`)
}

func (m *Maker) generateGoTable(w io.Writer, table *table) {
	m.generateGoTableInsert(w, table)
	m.generateGoTableSelect(w, table)
	m.generateGoTableUpdate(w, table)
}

func (m *Maker) generateGoTableInsert(w io.Writer, table *table) {
	// https://stackoverflow.com/questions/18100782/import-of-50k-records-in-mysql-gives-general-error-1390-prepared-statement-con
	const maxPlaceholderCount = 65535
	const maxMaxStructCount = 32

	fmt.Fprintf(w, "func Insert%[1]s(ctx context.Context, execer execer, values ...*%[1]s) error {", table.rawName)

	columns := make([]string, 0, len(table.columns))
	placeholders := make([]string, 0, len(table.columns))
	values := make([]string, 0, len(table.columns))
	for _, c := range table.columns {
		if c.autoIncr {
			continue
		}
		columns = append(columns, quote(c.name))
		placeholders = append(placeholders, "?")
		values = append(values, fmt.Sprintf("v.%s", c.rawName))
	}

	strPlaceholders := ", (" + strings.Join(placeholders, ", ") + ")"
	maxStructCount := maxPlaceholderCount / len(placeholders)
	if maxStructCount > maxMaxStructCount {
		maxStructCount = maxMaxStructCount
	}
	insert := "INSERT INTO " + quote(table.name) + " (" + strings.Join(columns, ", ") + ") VALUES" + " (" + strings.Join(placeholders, ", ") + ")"
	fmt.Fprintf(w, "const q = %q+\n%q\n", insert, strings.Repeat(strPlaceholders, maxStructCount-1))
	fmt.Fprintf(w, "const fieldCount = %d\n", len(placeholders))
	fmt.Fprintf(w, "const maxStructCount = %d\n", maxStructCount)

	fmt.Fprintf(w, `count := len(values)
	if count > maxStructCount {
		count = maxStructCount
	}
	args = make([]any, 0, count*fieldCount)
	for len(values) > 0 {
		i := len(values)
		if i > maxStructCount {
			i = maxStructCount
		}
		vals, rest := values[:i], values[i:]
		args = args[:0]
		for _, v := range vals {
			args = append(args, %s)
		}
		_, err := execer.ExecContext(ctx, q[:i*%d+%d], args...)
		if err != nil {
			return err
		}
		values = rest
	}
	return nil
}

`, strings.Join(values, ", "), len(strPlaceholders), len(insert)-len(strPlaceholders))
}

func (m *Maker) generateGoTableSelect(w io.Writer, table *table) {
	fields := make([]string, 0, len(table.columns))
	goFields := make([]string, 0, len(table.columns))
	params := make([]string, 0, len(table.primaryKey.columns))
	conditions := make([]string, 0, len(table.primaryKey.columns))
	for _, c := range table.columns {
		fields = append(fields, quote(c.name))
		goFields = append(goFields, "&v."+c.rawName)
		for _, key := range table.primaryKey.columns {
			if key == c.name {
				params = append(params, fmt.Sprintf("primaryKeys.%s", c.rawName))
				conditions = append(conditions, fmt.Sprintf("%s = ?", quote(c.name)))
			}
		}
	}

	sqlSelect := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s",
		strings.Join(fields, ", "),
		quote(table.name),
		strings.Join(conditions, " AND "),
	)
	fmt.Fprintf(w, "func Select%[1]s(ctx context.Context, queryer queryer, primaryKeys *%[1]s) (*%[1]s, error) {\n", table.rawName)
	fmt.Fprintf(w, "var v %s\n", table.rawName)
	fmt.Fprintf(w, "row := queryer.QueryRowContext(ctx, %q, %s)\n", sqlSelect, strings.Join(params, ", "))
	fmt.Fprintf(w, "if err := row.Scan(%s); err != nil {\n return nil, err \n}\n", strings.Join(goFields, ", "))
	fmt.Fprintf(w, "return &v, nil\n")
	fmt.Fprintf(w, "}\n\n")
}

func (m *Maker) generateGoTableUpdate(w io.Writer, table *table) {
	setFields := make([]string, 0, len(table.columns))
	goFields := make([]string, 0, len(table.columns))
	params := make([]string, 0, len(table.primaryKey.columns))
	conditions := make([]string, 0, len(table.primaryKey.columns))

LOOP:
	for _, c := range table.columns {
		for _, key := range table.primaryKey.columns {
			if key == c.name {
				params = append(params, fmt.Sprintf("value.%s", c.rawName))
				conditions = append(conditions, fmt.Sprintf("%s = ?", quote(c.name)))
				continue LOOP
			}
		}
		setFields = append(setFields, fmt.Sprintf("%s = ?", quote(c.name)))
		goFields = append(goFields, "value."+c.rawName)
	}

	update := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		quote(table.name),
		strings.Join(setFields, ", "),
		strings.Join(conditions, " AND "),
	)
	fmt.Fprintf(w, "func Update%[1]s(ctx context.Context, execer execer, value *%[1]s) error {\n", table.rawName)
	if len(setFields) != 0 {
		fmt.Fprintf(w, "_, err := execer.ExecContext(ctx, %q, %s, %s)\n", update, strings.Join(goFields, ", "), strings.Join(params, ", "))
		fmt.Fprintf(w, "return err\n")
	} else {
		fmt.Fprintf(w, "return nil\n")
	}
	fmt.Fprintf(w, "}\n\n")
}
