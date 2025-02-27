package myddlmaker

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/go-cmp/cmp"
)

type Foo1 struct {
	ID int32
}

func (*Foo1) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

type Foo2 struct {
	ID   int32  `ddl:",auto"`
	Name string `ddl:",comment='コメント',invisible"`
}

func (*Foo2) Table() string {
	return "foo2_customized"
}

func (*Foo2) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

func (*Foo2) Indexes() []*Index {
	return []*Index{
		NewIndex("idx_name", "name"),
	}
}

type Foo3 struct {
	ID   int32
	Name string
}

func (*Foo3) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

func (*Foo3) UniqueIndexes() []*UniqueIndex {
	return []*UniqueIndex{
		NewUniqueIndex("idx_name", "name"),
	}
}

type Foo4 struct {
	ID   int32
	Name string
}

func (*Foo4) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

func (*Foo4) ForeignKeys() []*ForeignKey {
	return []*ForeignKey{
		NewForeignKey("fk_foo1", []string{"id"}, "foo1", []string{"id"}),
	}
}

type Foo5 struct {
	ID   int32
	Name string
}

func (*Foo5) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

func (*Foo5) ForeignKeys() []*ForeignKey {
	return []*ForeignKey{
		NewForeignKey("fk_foo1", []string{"id"}, "foo1", []string{"id"}).OnUpdate(ForeignKeyOptionCascade).OnDelete(ForeignKeyOptionCascade),
	}
}

type Foo6 struct {
	ID    int32
	Name  string
	Email string
}

func (*Foo6) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

func (*Foo6) Indexes() []*Index {
	return []*Index{
		// Indexes with comments.
		NewIndex("idx_name", "name").Comment("an index\n\twith 'comment'"),
	}
}

func (*Foo6) UniqueIndexes() []*UniqueIndex {
	return []*UniqueIndex{
		// Indexes with comments.
		NewUniqueIndex("uniq_email", "email").Comment("a unique index\n\twith 'comment'"),
	}
}

type Foo7 struct {
	ID   int32
	Name string `ddl:",default='John Doe'"`
}

func (*Foo7) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

type Foo8 struct {
	ID   int32 `ddl:",auto"`
	Name string
}

func (*Foo8) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

func (*Foo8) Indexes() []*Index {
	return []*Index{
		NewIndex("idx_name", "name").Invisible(),
	}
}

type Foo9 struct {
	ID   int32  `ddl:",auto"`
	Name string `ddl:",charset=utf8,collate=utf8_bin"`
}

func (*Foo9) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

type Foo10 struct {
	ID   int32 `ddl:",auto"`
	Text string
}

func (*Foo10) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

func (*Foo10) FullTextIndexes() []*FullTextIndex {
	return []*FullTextIndex{
		NewFullTextIndex("idx_text", "text").WithParser("ngram").Comment("FULLTEXT INDEX"),
	}
}

type Foo11 struct {
	ID    int32  `ddl:",auto"`
	Point string `ddl:",type=GEOMETRY,srid=4326"`
}

func (*Foo11) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

func (*Foo11) SpatialIndexes() []*SpatialIndex {
	return []*SpatialIndex{
		NewSpatialIndex("idx_point", "point").Comment("SPATIAL INDEX"),
	}
}

type Foo12 struct {
	ID int32
}

func (*Foo12) Table() string {
	return "foo11"
}

func (*Foo12) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

type Foo13 struct {
	ID               int32 `ddl:"id"`
	DuplicatedColumn int32 `ddl:"id"`
}

func (*Foo13) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

type Foo14 struct {
	ID int32
}

func (*Foo14) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("unknown_column")
}

func (*Foo14) Indexes() []*Index {
	return []*Index{
		NewIndex("idx", "unknown_column"),
	}
}

func (*Foo14) UniqueIndexes() []*UniqueIndex {
	return []*UniqueIndex{
		NewUniqueIndex("uniq", "unknown_column"),
	}
}

type Foo15 struct {
	ID   int32
	Name string
}

func (*Foo15) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

func (*Foo15) Indexes() []*Index {
	return []*Index{
		NewIndex("idx_name", "name"),
	}
}

func (*Foo15) UniqueIndexes() []*UniqueIndex {
	return []*UniqueIndex{
		NewUniqueIndex("idx_name", "name"),
	}
}

func (*Foo15) FullTextIndexes() []*FullTextIndex {
	return []*FullTextIndex{
		NewFullTextIndex("idx_name", "name").WithParser("ngram").Comment("FULLTEXT INDEX"),
	}
}

func (*Foo15) SpatialIndexes() []*SpatialIndex {
	return []*SpatialIndex{
		NewSpatialIndex("idx_name", "name").Comment("SPATIAL INDEX"),
	}
}

type Foo16 struct {
	ID   int32
	Name string
}

func (*Foo16) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

func (*Foo16) ForeignKeys() []*ForeignKey {
	return []*ForeignKey{
		NewForeignKey("fk_duplicated", []string{"id"}, "foo16", []string{"id"}),
		NewForeignKey("fk_duplicated", []string{"id"}, "foo16", []string{"id"}),
	}
}

type Foo17 struct {
	ID int32
}

func (*Foo17) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

func (*Foo17) ForeignKeys() []*ForeignKey {
	return []*ForeignKey{
		NewForeignKey("fk_foo17", []string{"unknown_column"}, "unknown_table", []string{"id"}),
	}
}

type Foo18 struct {
	ID      int32
	Foo19ID int32
}

func (*Foo18) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

func (*Foo18) ForeignKeys() []*ForeignKey {
	return []*ForeignKey{
		// Foo18.Foo19ID is int32, but Foo19.ID is int64
		// it causes a type error
		NewForeignKey("fk_foo19", []string{"foo19_id"}, "foo19", []string{"id"}),
	}
}

type Foo19 struct {
	ID int64
}

func (*Foo19) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

type Foo20 struct {
	ID   int32 `ddl:",auto"`
	JSON JSON[struct {
		A string `json:"a"`
		B int    `json:"b"`
	}]
}

func (*Foo20) PrimaryKey() *PrimaryKey {
	return NewPrimaryKey("id")
}

func testMaker(t *testing.T, structs []any, ddl string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m, err := New(&Config{
		DB: &DBConfig{
			Engine:  "InnoDB",
			Charset: "utf8mb4",
			Collate: "utf8mb4_bin",
		},
	})
	if err != nil {
		t.Fatalf("failed to initialize Maker: %v", err)
	}

	m.AddStructs(structs...)

	var buf bytes.Buffer
	if err := m.Generate(&buf); err != nil {
		t.Fatalf("failed to generate ddl: %v", err)
	}

	got := buf.String()
	if diff := cmp.Diff(ddl, got); diff != "" {
		t.Errorf("ddl is not match: (-want/+got)\n%s", diff)
	}

	db, ok := setupDatabase(ctx, t)
	if !ok {
		return
	}

	// check the ddl syntax
	if _, err := db.ExecContext(ctx, got); err != nil {
		t.Errorf("failed to execute %q: %v", got, err)
	}
}

func setupDatabase(ctx context.Context, t testing.TB) (db *sql.DB, ok bool) {
	// check the ddl syntax with MySQL Server
	user := os.Getenv("MYSQL_TEST_USER")
	pass := os.Getenv("MYSQL_TEST_PASS")
	addr := os.Getenv("MYSQL_TEST_ADDR")
	if user == "" || pass == "" || addr == "" {
		return nil, false
	}

	// connect to the server
	cfg := mysql.NewConfig()
	cfg.User = user
	cfg.Passwd = pass
	cfg.Addr = addr
	db0, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db0.Close()

	// create a new database
	var buf2 [4]byte
	_, err = rand.Read(buf2[:])
	if err != nil {
		t.Fatal(err)
	}
	dbName := fmt.Sprintf("myddlmaker_%x", buf2[:])
	_, err = db0.ExecContext(ctx, "CREATE DATABASE "+dbName)
	if err != nil {
		t.Fatalf("failed to create database %q: %v", dbName, err)
	}

	cfg.DBName = dbName
	cfg.MultiStatements = true
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() {
		db0.ExecContext(ctx, "DROP DATABASE "+dbName)
		db.Close()
	})
	return db, true
}

func testMakerError(t *testing.T, structs []any, wantErr []string) {
	t.Helper()

	m, err := New(&Config{
		DB: &DBConfig{
			Engine:  "InnoDB",
			Charset: "utf8mb4",
			Collate: "utf8mb4_bin",
		},
	})
	if err != nil {
		t.Fatalf("failed to initialize Maker: %v", err)
	}

	m.AddStructs(structs...)

	var buf bytes.Buffer
	err = m.Generate(&buf)
	if err == nil {
		t.Error("want some error, but not")
		return
	}

	var errs *validationError
	if !errors.As(err, &errs) {
		t.Errorf("unexpected error type: %T", err)
	}

	if diff := cmp.Diff(wantErr, errs.errs); diff != "" {
		t.Errorf("unexpected errors (-want/+got):\n%s", diff)
	}
}

func TestMaker_Generate(t *testing.T) {
	testMaker(t, []any{&Foo1{}}, "SET foreign_key_checks=0;\n\n"+
		"DROP TABLE IF EXISTS `foo1`;\n\n"+
		"CREATE TABLE `foo1` (\n"+
		"    `id` INTEGER NOT NULL,\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n"+
		"SET foreign_key_checks=1;\n")

	testMaker(t, []any{&Foo2{}}, "SET foreign_key_checks=0;\n\n"+
		"DROP TABLE IF EXISTS `foo2_customized`;\n\n"+
		"CREATE TABLE `foo2_customized` (\n"+
		"    `id` INTEGER NOT NULL AUTO_INCREMENT,\n"+
		"    `name` VARCHAR(191) NOT NULL INVISIBLE COMMENT '\\'コメント\\'',\n"+
		"    INDEX `idx_name` (`name`),\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n"+
		"SET foreign_key_checks=1;\n")

	testMaker(t, []any{&Foo3{}}, "SET foreign_key_checks=0;\n\n"+
		"DROP TABLE IF EXISTS `foo3`;\n\n"+
		"CREATE TABLE `foo3` (\n"+
		"    `id` INTEGER NOT NULL,\n"+
		"    `name` VARCHAR(191) NOT NULL,\n"+
		"    UNIQUE `idx_name` (`name`),\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n"+
		"SET foreign_key_checks=1;\n")

	testMaker(t, []any{&Foo1{}, &Foo4{}}, "SET foreign_key_checks=0;\n\n"+
		"DROP TABLE IF EXISTS `foo1`;\n\n"+
		"CREATE TABLE `foo1` (\n"+
		"    `id` INTEGER NOT NULL,\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n\n"+
		"DROP TABLE IF EXISTS `foo4`;\n\n"+
		"CREATE TABLE `foo4` (\n"+
		"    `id` INTEGER NOT NULL,\n"+
		"    `name` VARCHAR(191) NOT NULL,\n"+
		"    CONSTRAINT `fk_foo1` FOREIGN KEY (`id`) REFERENCES `foo1` (`id`),\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n"+
		"SET foreign_key_checks=1;\n")

	testMaker(t, []any{&Foo5{}, &Foo1{}}, "SET foreign_key_checks=0;\n\n"+
		"DROP TABLE IF EXISTS `foo5`;\n\n"+
		"CREATE TABLE `foo5` (\n"+
		"    `id` INTEGER NOT NULL,\n"+
		"    `name` VARCHAR(191) NOT NULL,\n"+
		"    CONSTRAINT `fk_foo1` FOREIGN KEY (`id`) REFERENCES `foo1` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n\n"+
		"DROP TABLE IF EXISTS `foo1`;\n\n"+
		"CREATE TABLE `foo1` (\n"+
		"    `id` INTEGER NOT NULL,\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n"+
		"SET foreign_key_checks=1;\n")

	testMaker(t, []any{&Foo6{}}, "SET foreign_key_checks=0;\n\n"+
		"DROP TABLE IF EXISTS `foo6`;\n\n"+
		"CREATE TABLE `foo6` (\n"+
		"    `id` INTEGER NOT NULL,\n"+
		"    `name` VARCHAR(191) NOT NULL,\n"+
		"    `email` VARCHAR(191) NOT NULL,\n"+
		"    INDEX `idx_name` (`name`) COMMENT 'an index\\n\\twith \\'comment\\'',\n"+
		"    UNIQUE `uniq_email` (`email`) COMMENT 'a unique index\\n\\twith \\'comment\\'',\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n"+
		"SET foreign_key_checks=1;\n")

	testMaker(t, []any{&Foo7{}}, "SET foreign_key_checks=0;\n\n"+
		"DROP TABLE IF EXISTS `foo7`;\n\n"+
		"CREATE TABLE `foo7` (\n"+
		"    `id` INTEGER NOT NULL,\n"+
		"    `name` VARCHAR(191) NOT NULL DEFAULT 'John Doe',\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n"+
		"SET foreign_key_checks=1;\n")

	// invisible index
	testMaker(t, []any{&Foo8{}}, "SET foreign_key_checks=0;\n\n"+
		"DROP TABLE IF EXISTS `foo8`;\n\n"+
		"CREATE TABLE `foo8` (\n"+
		"    `id` INTEGER NOT NULL AUTO_INCREMENT,\n"+
		"    `name` VARCHAR(191) NOT NULL,\n"+
		"    INDEX `idx_name` (`name`) INVISIBLE,\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n"+
		"SET foreign_key_checks=1;\n")

	// charset and collate
	testMaker(t, []any{&Foo9{}}, "SET foreign_key_checks=0;\n\n"+
		"DROP TABLE IF EXISTS `foo9`;\n\n"+
		"CREATE TABLE `foo9` (\n"+
		"    `id` INTEGER NOT NULL AUTO_INCREMENT,\n"+
		"    `name` VARCHAR(191) CHARACTER SET utf8 COLLATE utf8_bin NOT NULL,\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n"+
		"SET foreign_key_checks=1;\n")

	// FULLTEXT INDEX
	testMaker(t, []any{&Foo10{}}, "SET foreign_key_checks=0;\n\n"+
		"DROP TABLE IF EXISTS `foo10`;\n\n"+
		"CREATE TABLE `foo10` (\n"+
		"    `id` INTEGER NOT NULL AUTO_INCREMENT,\n"+
		"    `text` VARCHAR(191) NOT NULL,\n"+
		"    FULLTEXT INDEX `idx_text` (`text`) WITH PARSER ngram COMMENT 'FULLTEXT INDEX',\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n"+
		"SET foreign_key_checks=1;\n")

	// SPATIAL INDEX
	testMaker(t, []any{&Foo11{}}, "SET foreign_key_checks=0;\n\n"+
		"DROP TABLE IF EXISTS `foo11`;\n\n"+
		"CREATE TABLE `foo11` (\n"+
		"    `id` INTEGER NOT NULL AUTO_INCREMENT,\n"+
		"    `point` GEOMETRY NOT NULL,\n"+
		"    SPATIAL INDEX `idx_point` (`point`) COMMENT 'SPATIAL INDEX',\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n"+
		"SET foreign_key_checks=1;\n")

	// JSON
	testMaker(t, []any{&Foo20{}}, "SET foreign_key_checks=0;\n\n"+
		"DROP TABLE IF EXISTS `foo20`;\n\n"+
		"CREATE TABLE `foo20` (\n"+
		"    `id` INTEGER NOT NULL AUTO_INCREMENT,\n"+
		"    `json` JSON NOT NULL,\n"+
		"    PRIMARY KEY (`id`)\n"+
		") ENGINE=InnoDB DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin;\n\n"+
		"SET foreign_key_checks=1;\n")

	testMakerError(t, []any{&Foo11{}, &Foo12{}}, []string{
		`duplicated name of table: "foo11"`,
	})

	testMakerError(t, []any{&Foo13{}}, []string{
		`table "foo13": duplicated name of column: "id"`,
	})

	testMakerError(t, []any{&Foo14{}}, []string{
		`table "foo14", primary key: column "unknown_column" not found`,
		`table "foo14", index "idx": column "unknown_column" not found`,
		`table "foo14", unique index "uniq": column "unknown_column" not found`,
	})

	testMakerError(t, []any{&Foo15{}}, []string{
		`table "foo15": duplicated name of index: "idx_name"`,
		`table "foo15": duplicated name of index: "idx_name"`,
		`table "foo15": duplicated name of index: "idx_name"`,
	})

	testMakerError(t, []any{&Foo16{}}, []string{
		`table "foo16": duplicated name of foreign key constraint: "fk_duplicated"`,
	})

	testMakerError(t, []any{&Foo17{}}, []string{
		`table "foo17", foreign key "fk_foo17": column "unknown_column" not found`,
		`table "foo17", foreign key "fk_foo17": referenced table "unknown_table" not found`,
	})

	testMakerError(t, []any{&Foo18{}, &Foo19{}}, []string{
		`table "foo18", foreign key "fk_foo19": index required on table "foo18"`,
		`table "foo18", foreign key "fk_foo19": column "foo19_id" and referenced column "foo19"."id" type mismatch`,
	})
}

func TestMaker_GenerateGo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fn := func(dir string) func(t *testing.T) {
		// gen generates 'scheme.sql' and 'scheme_gen.go'
		gen := func(t *testing.T) error {
			ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()

			var buf bytes.Buffer
			args := []string{"run", "-tags", "myddlmaker", filepath.Join("gen", "main.go")}
			cmd := exec.CommandContext(ctx, goTool(), args...)
			cmd.Stdout = &buf
			cmd.Stderr = &buf
			cmd.Dir = dir
			if err := cmd.Run(); err != nil {
				t.Errorf("failed to generate: %v, output:\n%s", err, buf.String())
				return err
			}
			return nil
		}

		runTests := func(t *testing.T) error {
			ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()

			var buf bytes.Buffer
			args := []string{"test"}
			cmd := exec.CommandContext(ctx, goTool(), args...)
			cmd.Stdout = &buf
			cmd.Stderr = &buf
			cmd.Dir = dir
			if err := cmd.Run(); err != nil {
				t.Errorf("failed to run test: %v, output:\n%s", err, buf.String())
				return err
			}
			return nil
		}
		return func(t *testing.T) {
			if err := gen(t); err != nil {
				return
			}

			// check the ddl syntax with MySQL Server
			user := os.Getenv("MYSQL_TEST_USER")
			pass := os.Getenv("MYSQL_TEST_PASS")
			addr := os.Getenv("MYSQL_TEST_ADDR")
			if user != "" && pass != "" && addr != "" {
				ddl, err := os.ReadFile(filepath.Join(dir, "schema.sql"))
				if err != nil {
					t.Errorf("failed read schema.sql: %v", err)
					return
				}

				// connect to the server
				cfg := mysql.NewConfig()
				cfg.User = user
				cfg.Passwd = pass
				cfg.Addr = addr
				db0, err := sql.Open("mysql", cfg.FormatDSN())
				if err != nil {
					t.Fatalf("failed to open db: %v", err)
				}
				defer db0.Close()

				// create a new database
				var buf2 [4]byte
				_, err = rand.Read(buf2[:])
				if err != nil {
					t.Fatal(err)
				}
				dbName := fmt.Sprintf("myddlmaker_%x", buf2[:])
				_, err = db0.ExecContext(ctx, "CREATE DATABASE "+dbName)
				if err != nil {
					t.Fatalf("failed to create database %q: %v", dbName, err)
				}
				defer db0.ExecContext(ctx, "DROP DATABASE "+dbName)
				t.Setenv("MYSQL_TEST_DB", dbName)

				// apply the ddl
				cfg.DBName = dbName
				cfg.MultiStatements = true
				db, err := sql.Open("mysql", cfg.FormatDSN())
				if err != nil {
					t.Fatalf("failed to open db: %v", err)
				}
				defer db.Close()
				if _, err := db.ExecContext(ctx, string(ddl)); err != nil {
					t.Errorf("failed to execute %q: %v", string(ddl), err)
				}
			}

			if err := runTests(t); err != nil {
				return
			}
		}
	}
	dirs, err := filepath.Glob("./testdata/*")
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range dirs {
		stat, err := os.Stat(dir)
		if err != nil {
			t.Error(err)
			continue
		}
		if !stat.IsDir() {
			continue
		}
		t.Run(dir, fn(dir))
	}
}

// goTool reports the path of the go tool to use to run the tests.
// If possible, use the same Go used to run run.go, otherwise
// fallback to the go version found in the PATH.
func goTool() string {
	var exeSuffix string
	if runtime.GOOS == "windows" {
		exeSuffix = ".exe"
	}
	path := filepath.Join(runtime.GOROOT(), "bin", "go"+exeSuffix)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	// Just run "go" from PATH
	return "go"
}
