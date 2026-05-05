package tmppg

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Instance struct {
	connString string
	log        *slog.Logger
}

func NewInstance(connString string) *Instance {
	return &Instance{
		connString: connString,
		log:        slog.Default(),
	}
}

type DatabaseOption func(pg *pgxpool.Config) error

func (i *Instance) WithDatabase(ctx context.Context, fn func(pool *pgxpool.Pool) error, opts ...DatabaseOption) (err error) {
	var conn *pgx.Conn
	conn, err = pgx.Connect(ctx, i.connString+" dbname=postgres")
	if err != nil {
		return fmt.Errorf("connect to admin database: %w", err)
	}
	defer conn.Close(ctx)

	dbname := fmt.Sprintf("test%d", rand.Uint32())
	i.log.Info("creating database", slog.String("name", dbname))
	_, err = conn.Exec(ctx, "CREATE DATABASE "+dbname)
	if err != nil {
		return fmt.Errorf("create database %q: %w", dbname, err)
	}

	// run database removal deferred, so the database also gets removed on
	// runtime.Goexit() and t.FailNow()
	defer func() {
		i.log.Info("dropping database", slog.String("name", dbname))
		_, dropError := conn.Exec(ctx, "DROP DATABASE "+dbname+" WITH (FORCE)")
		if dropError != nil {
			if err == nil {
				err = fmt.Errorf("drop database %q: %w", dbname, dropError)
			} else {
				err = fmt.Errorf("drop database %q: %w; previous error: %w", dbname, dropError, err)
			}
		}
	}()

	dsn := i.connString + " dbname=" + dbname
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("parse DSN %q: %w", dsn, err)
	}
	for i, opt := range opts {
		if err := opt(cfg); err != nil {
			return fmt.Errorf("appy option %d: %w", i, err)
		}
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect to database %q: %w", dbname, err)
	}
	defer pool.Close()

	if err = fn(pool); err != nil {
		return fmt.Errorf("in function: %w", err)
	}

	return nil
}

func (i *Instance) WithDatabaseSchema(ctx context.Context, schemaSQL string, fn func(pool *pgxpool.Pool) error, opts ...DatabaseOption) error {
	return i.WithDatabase(ctx, func(pool *pgxpool.Pool) error {
		// run DDL on its own connection in case it does any weird stuff like `SET search_path`
		// closing the connection afterwards ensures that `SET`s are discarded
		err := pool.AcquireFunc(ctx, func(conn *pgxpool.Conn) error {
			_, err := conn.Exec(ctx, schemaSQL)
			if err != nil {
				return err
			}
			err = conn.Conn().Close(ctx)
			return err
		})
		if err != nil {
			return fmt.Errorf("create schema: %w", err)
		}
		return fn(pool)
	}, opts...)
}
