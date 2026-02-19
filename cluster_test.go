package tmppg

import (
	"testing"

	"github.com/authenticvision/tmppg/internal/errutil"
	"github.com/stretchr/testify/require"
)

func TestCluster(t *testing.T) {
	r := require.New(t)
	pg := NewPostgres(WithoutSync())
	dir := t.TempDir()
	c, err := OpenCluster(pg, dir)
	r.Error(err)
	r.Nil(c)
	r.True(errutil.IsType[*UninitializedError](err))
	c, err = OpenOrCreateCluster(pg, dir)
	r.NoError(err)
	r.NotNil(c)
	c, err = CreateCluster(pg, dir)
	r.ErrorContains(err, "cluster dir exists and is not empty")
	r.Nil(c)
}
