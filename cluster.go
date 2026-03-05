package tmppg

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/authenticvision/tmppg/internal/errutil"
	"github.com/authenticvision/tmppg/internal/logutil"
)

type Cluster struct {
	pg        *Postgres
	dir       string
	cmd       *exec.Cmd
	exitErrCh chan error
}

type UninitializedError struct {
	err error
}

func (e *UninitializedError) Error() string {
	return fmt.Sprintf("cluster uninitialized: %s", e.err.Error())
}

func (e *UninitializedError) Unwrap() error {
	return e.err
}

func OpenOrCreateCluster(pg *Postgres, dir string) (*Cluster, error) {
	if c, err := OpenCluster(pg, dir); err == nil {
		return c, nil
	} else if !errutil.IsType[*UninitializedError](err) {
		return nil, err
	}
	return CreateCluster(pg, dir)
}

func OpenCluster(pg *Postgres, dir string) (*Cluster, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, &UninitializedError{err}
	} else if err != nil {
		return nil, fmt.Errorf("read dir cluster: %w", err)
	} else if len(entries) == 0 {
		return nil, &UninitializedError{errors.New("cluster dir exists but is empty")}
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("absolute path for postgres dir: %w", err)
	}
	return &Cluster{
		pg:  pg,
		dir: dir,
	}, nil
}

func CreateCluster(pg *Postgres, dir string) (*Cluster, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, fmt.Errorf("mkdir cluster: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("read dir cluster: %w", err)
	} else if len(entries) > 0 {
		return nil, fmt.Errorf("cluster dir exists and is not empty")
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("absolute path for postgres dir: %w", err)
	}
	err = pg.initDB(dir)
	if err != nil {
		return nil, fmt.Errorf("initdb: %w", err)
	}
	return &Cluster{
		pg:  pg,
		dir: dir,
	}, nil
}

// pg_isready exit codes
const (
	pqPingReject     = 1
	pqPingNoResponse = 2
	pqPingNoAttempt  = 3
)

func (c *Cluster) Start() error {
	pgCmd, err := c.pg.start(c.dir)
	if err != nil {
		return fmt.Errorf("start cluster: %w", err)
	}
	c.cmd = pgCmd

	c.exitErrCh = make(chan error, 1)
	go func() {
		c.exitErrCh <- pgCmd.Wait()
		close(c.exitErrCh)
	}()

	for {
		select {
		case err := <-c.exitErrCh:
			return fmt.Errorf("postgres exited unexpectedly: %w", err)
		case <-time.After(100 * time.Millisecond):
		}
		cmd := c.pg.makeCmd("pg_isready", "-q", "-h", c.dir, "-d", "postgres")
		err := cmd.Run()
		var exitErr *exec.ExitError
		if err == nil {
			break
		} else if errors.As(err, &exitErr) {
			if exitErr.ExitCode() == pqPingReject || exitErr.ExitCode() == pqPingNoResponse {
				slog.Debug("waiting for PostgreSQL to be ready")
			} else {
				if stopErr := c.Stop(); stopErr != nil {
					slog.Error("failed to stop cluster", logutil.Err(stopErr))
				}
				return fmt.Errorf("pg_isready: %w", err)
			}
		}
	}
	return nil
}

func (c *Cluster) Stop() error {
	if c.cmd == nil {
		return fmt.Errorf("cluster is not running")
	}

	err := c.cmd.Process.Signal(syscall.SIGINT)
	if err != nil {
		return fmt.Errorf("terminate cluster: %w", err)
	}
	err = <-c.exitErrCh
	if errutil.IsType[*exec.ExitError](err) {
		return fmt.Errorf("postgres exited with error: %w", err)
	} else if err != nil {
		return fmt.Errorf("failed to wait for postgres: %w", err)
	}
	return nil
}

func (c *Cluster) URL() *url.URL {
	return &url.URL{
		Scheme: "postgres",
		Path:   "/postgres",
		RawQuery: url.Values{
			"host":    {c.dir},
			"sslmode": {"disable"},
		}.Encode(),
	}
}
