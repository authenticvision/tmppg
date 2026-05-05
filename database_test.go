package tmppg

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ERRCODE_INVALID_CATALOG_NAME = "3D000"

func TestInstance_WithDatabase_Cleanup(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)
	err := WithPostgresql(func(socketDir string) error {
		pg := NewInstance("host=" + socketDir)
		var dbname string
		err := pg.WithDatabase(t.Context(), func(pool *pgxpool.Pool) error {
			row := pool.QueryRow(t.Context(), "SELECT current_database();")
			err := row.Scan(&dbname)
			r.NoError(err)
			return errors.New("test error")
		})
		a.Error(err)
		_, err = pgx.Connect(t.Context(), "host="+socketDir+" dbname="+dbname)
		var pgError *pgconn.PgError
		a.ErrorAs(err, &pgError)
		a.Equal(ERRCODE_INVALID_CATALOG_NAME, pgError.Code)
		return nil
	})
	r.NoError(err)
}

func TestInstance_WithDatabase_Panic(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)
	err := WithPostgresql(func(socketDir string) error {
		pg := NewInstance("host=" + socketDir)
		var dbname string
		r.PanicsWithValue("test panic", func() {
			_ = pg.WithDatabase(t.Context(), func(pool *pgxpool.Pool) error {
				dbname = pool.Config().ConnConfig.Database
				panic("test panic")
			})
		})
		_, err := pgx.Connect(t.Context(), "host="+socketDir+" dbname="+dbname)
		var pgError *pgconn.PgError
		a.ErrorAs(err, &pgError)
		a.Equal(ERRCODE_INVALID_CATALOG_NAME, pgError.Code)
		return nil
	})
	r.NoError(err)
}

func TestInstance_WithDatabase_RuntimeGoexit(t *testing.T) {
	t.Skip("this test will always fail. execute manually to verify cleanup behavior.")
	t.Parallel()
	_ = WithPostgresql(func(socketDir string) error {
		pg := NewInstance("host=" + socketDir)
		_ = pg.WithDatabase(t.Context(), func(pool *pgxpool.Pool) error {
			println("exiting now via t.FailNow()")
			t.FailNow()
			return nil
		})
		return nil
	})
}

const schemaSQL = `
CREATE TABLE test (
	id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
	name TEXT NOT NULL
);

INSERT INTO test (name) VALUES ('test');
`

func TestInstance_WithDatabaseSchema(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	a := assert.New(t)
	err := WithPostgresql(func(socketDir string) error {
		pg := NewInstance("host=" + socketDir)
		err := pg.WithDatabaseSchema(t.Context(), schemaSQL, func(pool *pgxpool.Pool) error {
			var id int64
			var name, dbname string
			row := pool.QueryRow(t.Context(), "SELECT id, name, current_database() FROM test;")
			err := row.Scan(&id, &name, &dbname)
			if err != nil {
				return err
			}
			a.Equal(int64(1), id)
			a.Equal("test", name)
			a.Contains(dbname, "test")
			return err
		})
		return err
	})
	r.NoError(err)
}

func TestInstance_WithDatabase_Options(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	called := false
	opt := func(cfg *pgxpool.Config) error {
		cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			called = true
			return nil
		}
		return nil
	}
	err := WithPostgresql(func(socketDir string) error {
		pg := NewInstance("host=" + socketDir)
		return pg.WithDatabaseSchema(t.Context(), `SELECT 1`, func(pool *pgxpool.Pool) error {
			return pool.AcquireFunc(t.Context(), func(conn *pgxpool.Conn) error {
				return conn.Ping(t.Context())
			})
		}, opt)
	})
	r.NoError(err)
	r.True(called, "AfterConnect called")
}
