// Copyright 2018-2022 Burak Sezer
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package olric

import (
	"context"
	"github.com/buraksezer/olric/config"
	"log"
	"os"
	"testing"
	"time"

	"github.com/buraksezer/olric/stats"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestClusterClient_Ping(t *testing.T) {
	cluster := newTestOlricCluster(t)
	cluster.addMember(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	err = c.Ping(ctx, db.rt.This().String())
	require.NoError(t, err)
}

func TestClusterClient_PingWithMessage(t *testing.T) {
	cluster := newTestOlricCluster(t)
	cluster.addMember(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	message := "Olric is the best!"
	result, err := c.PingWithMessage(ctx, db.rt.This().String(), message)
	require.NoError(t, err)
	require.Equal(t, message, result)
}

func TestClusterClient_RoutingTable(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	rt, err := c.RoutingTable(ctx)
	require.NoError(t, err)

	require.Len(t, rt, int(db.config.PartitionCount))
}

func TestClusterClient_Put(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	err = dm.Put(ctx, "mykey", "myvalue")
	require.NoError(t, err)
}

func TestClusterClient_Get(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	err = dm.Put(ctx, "mykey", "myvalue")
	require.NoError(t, err)

	gr, err := dm.Get(ctx, "mykey")
	require.NoError(t, err)

	res, err := gr.String()
	require.NoError(t, err)

	require.Equal(t, res, "myvalue")
}

func TestClusterClient_Delete(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	err = dm.Put(ctx, "mykey", "myvalue")
	require.NoError(t, err)

	err = dm.Delete(ctx, "mykey")
	require.NoError(t, err)

	_, err = dm.Get(ctx, "mykey")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestClusterClient_Destroy(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	err = dm.Put(ctx, "mykey", "myvalue")
	require.NoError(t, err)

	err = dm.Destroy(ctx)
	require.NoError(t, err)

	_, err = dm.Get(ctx, "mykey")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestClusterClient_Incr(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	var errGr errgroup.Group
	for i := 0; i < 10; i++ {
		errGr.Go(func() error {
			_, err = dm.Incr(ctx, "mykey", 1)
			return err
		})
	}

	require.NoError(t, errGr.Wait())

	result, err := dm.Incr(ctx, "mykey", 1)
	require.NoError(t, err)
	require.Equal(t, 11, result)
}

func TestClusterClient_Decr(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	err = dm.Put(ctx, "mykey", 11)
	require.NoError(t, err)

	var errGr errgroup.Group
	for i := 0; i < 10; i++ {
		errGr.Go(func() error {
			_, err = dm.Decr(ctx, "mykey", 1)
			return err
		})
	}

	require.NoError(t, errGr.Wait())

	result, err := dm.Decr(ctx, "mykey", 1)
	require.NoError(t, err)
	require.Equal(t, 0, result)
}

func TestClusterClient_GetPut(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	gr, err := dm.GetPut(ctx, "mykey", "myvalue")
	require.NoError(t, err)
	require.Nil(t, gr)

	gr, err = dm.GetPut(ctx, "mykey", "myvalue-2")
	require.NoError(t, err)

	value, err := gr.String()
	require.NoError(t, err)
	require.Equal(t, "myvalue", value)
}

func TestClusterClient_Expire(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	err = dm.Put(ctx, "mykey", "myvalue")
	require.NoError(t, err)

	err = dm.Expire(ctx, "mykey", time.Millisecond)
	require.NoError(t, err)

	<-time.After(time.Millisecond)

	_, err = dm.Get(ctx, "mykey")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestClusterClient_Lock_Unlock(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	lx, err := dm.Lock(ctx, "lock.foo.key", time.Second)
	require.NoError(t, err)

	err = lx.Unlock(ctx)
	require.NoError(t, err)
}

func TestClusterClient_Lock_Lease(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	lx, err := dm.Lock(ctx, "lock.foo.key", time.Second)
	require.NoError(t, err)

	err = lx.Lease(ctx, time.Millisecond)
	require.NoError(t, err)

	<-time.After(time.Millisecond)

	err = lx.Unlock(ctx)
	require.ErrorIs(t, err, ErrNoSuchLock)
}

func TestClusterClient_Lock_ErrLockNotAcquired(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	_, err = dm.Lock(ctx, "lock.foo.key", time.Second)
	require.NoError(t, err)

	_, err = dm.Lock(ctx, "lock.foo.key", time.Millisecond)
	require.ErrorIs(t, err, ErrLockNotAcquired)
}

func TestClusterClient_LockWithTimeout(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	lx, err := dm.LockWithTimeout(ctx, "lock.foo.key", time.Hour, time.Second)
	require.NoError(t, err)

	err = lx.Unlock(ctx)
	require.NoError(t, err)
}

func TestClusterClient_LockWithTimeout_ErrNoSuchLock(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	lx, err := dm.LockWithTimeout(ctx, "lock.foo.key", time.Millisecond, time.Second)
	require.NoError(t, err)

	<-time.After(time.Millisecond)

	err = lx.Unlock(ctx)
	require.ErrorIs(t, err, ErrNoSuchLock)
}

func TestClusterClient_LockWithTimeout_Then_Lease(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	lx, err := dm.LockWithTimeout(ctx, "lock.foo.key", 50*time.Millisecond, time.Second)
	require.NoError(t, err)

	// Expand its timeout value
	err = lx.Lease(ctx, time.Hour)
	require.NoError(t, err)

	<-time.After(100 * time.Millisecond)

	_, err = dm.Lock(ctx, "lock.foo.key", time.Millisecond)
	require.ErrorIs(t, err, ErrLockNotAcquired)
}

func TestClusterClient_LockWithTimeout_ErrLockNotAcquired(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	_, err = dm.LockWithTimeout(ctx, "lock.foo.key", time.Hour, time.Second)
	require.NoError(t, err)

	_, err = dm.Lock(ctx, "lock.foo.key", time.Millisecond)
	require.Equal(t, err, ErrLockNotAcquired)
}

func TestClusterClient_Put_Ex(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	err = dm.Put(ctx, "mykey", "myvalue", EX(time.Second))
	require.NoError(t, err)

	<-time.After(time.Second)

	_, err = dm.Get(ctx, "mykey")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestClusterClient_Put_PX(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	err = dm.Put(ctx, "mykey", "myvalue", PX(time.Millisecond))
	require.NoError(t, err)

	<-time.After(time.Millisecond)

	_, err = dm.Get(ctx, "mykey")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestClusterClient_Put_EXAT(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	err = dm.Put(ctx, "mykey", "myvalue", EXAT(time.Duration(time.Now().Add(time.Second).UnixNano())))
	require.NoError(t, err)

	<-time.After(time.Second)

	_, err = dm.Get(ctx, "mykey")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestClusterClient_Put_PXAT(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	err = dm.Put(ctx, "mykey", "myvalue", PXAT(time.Duration(time.Now().Add(time.Millisecond).UnixNano())))
	require.NoError(t, err)

	<-time.After(time.Millisecond)

	_, err = dm.Get(ctx, "mykey")
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestClusterClient_Put_NX(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	err = dm.Put(ctx, "mykey", "myvalue")
	require.NoError(t, err)

	err = dm.Put(ctx, "mykey", "myvalue-2", NX())
	require.ErrorIs(t, err, ErrKeyFound)

	gr, err := dm.Get(ctx, "mykey")
	require.NoError(t, err)

	value, err := gr.String()
	require.NoError(t, err)
	require.Equal(t, "myvalue", value)
}

func TestClusterClient_Put_XX(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	dm, err := c.NewDMap("mydmap")
	require.NoError(t, err)

	err = dm.Put(ctx, "mykey", "myvalue-2", XX())
	require.ErrorIs(t, err, ErrKeyNotFound)
}

func TestClusterClient_Stats(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	var empty stats.Stats
	s, err := c.Stats(ctx)
	require.NoError(t, err)
	require.Nil(t, s.Runtime)
	require.NotEqual(t, empty, s)
}

func TestClusterClient_Stats_CollectRuntime(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()
	c, err := NewClusterClient([]string{db.name})
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	var empty stats.Stats
	s, err := c.Stats(ctx, CollectRuntime())
	require.NoError(t, err)
	require.NotNil(t, s.Runtime)
	require.NotEqual(t, empty, s)
}

func TestClusterClient_Set_Options(t *testing.T) {
	cluster := newTestOlricCluster(t)
	db := cluster.addMember(t)

	ctx := context.Background()

	lg := log.New(os.Stderr, "logger: ", log.Lshortfile)
	cfg := config.NewClient()
	c, err := NewClusterClient([]string{db.name}, WithConfig(cfg), WithLogger(lg))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close(ctx))
	}()

	require.Equal(t, cfg, c.config.config)
	require.Equal(t, lg, c.config.logger)
}
