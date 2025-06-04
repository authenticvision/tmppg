# Temporary PostgreSQL instances for Go

tmppg locally spawns a new PostgreSQL database cluster for you to create databases in, mostly for tests. Since initializing a database cluster is relatively expensive, [it can be done in a wrapper program](https://michael.stapelberg.ch/posts/2024-11-19-testing-with-go-and-postgresql-ephemeral-dbs/#sharing-one-postgresql-among-all-tests): `go run github.com/authenticvision/tmppg/cmd/wrapper@latest -- go test ./...`

tmppg is a reimplementation of [github.com/stapelberg/postgrestest](https://github.com/stapelberg/postgrestest) with different opinions:
- Run `postgres` directly instead of via `pg_ctl`
- Pass config to `postgres` via CLI arguments instead of using a config file
- No Windows support
- An effectively worse startup check
- Use `pgx` instead of `database/sql`
