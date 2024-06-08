package key

import (
	"testing"
	"time"

	"github.com/nalgeon/redka/internal/redis"
	"github.com/nalgeon/redka/internal/testx"
)

func TestPExpireParse(t *testing.T) {
	tests := []struct {
		cmd string
		key string
		ttl time.Duration
		err error
	}{
		{
			cmd: "pexpire",
			key: "",
			ttl: 0,
			err: redis.ErrInvalidArgNum,
		},
		{
			cmd: "pexpire name",
			key: "",
			ttl: 0,
			err: redis.ErrInvalidArgNum,
		},
		{
			cmd: "pexpire name 5000",
			key: "name",
			ttl: 5000 * time.Millisecond,
			err: nil,
		},
		{
			cmd: "pexpire name age",
			key: "",
			ttl: 0,
			err: redis.ErrInvalidInt,
		},
		{
			cmd: "pexpire name 100 age 100",
			key: "",
			ttl: 0,
			err: redis.ErrSyntaxError,
		},
	}

	parse := func(b redis.BaseCmd) (Expire, error) {
		return ParseExpire(b, 1)
	}

	for _, test := range tests {
		t.Run(test.cmd, func(t *testing.T) {
			cmd, err := redis.Parse(parse, test.cmd)
			testx.AssertEqual(t, err, test.err)
			if err == nil {
				testx.AssertEqual(t, cmd.key, test.key)
				testx.AssertEqual(t, cmd.ttl, test.ttl)
			} else {
				testx.AssertEqual(t, cmd, Expire{})
			}
		})
	}
}

func TestPExpireExec(t *testing.T) {
	parse := func(b redis.BaseCmd) (Expire, error) {
		return ParseExpire(b, 1)
	}

	t.Run("create pexpire", func(t *testing.T) {
		db, red := getDB(t)
		defer db.Close()
		_ = db.Str().Set("name", "alice")

		cmd := redis.MustParse(parse, "pexpire name 60000")
		conn := redis.NewFakeConn()
		res, err := cmd.Run(conn, red)
		testx.AssertNoErr(t, err)
		testx.AssertEqual(t, res, true)
		testx.AssertEqual(t, conn.Out(), "1")

		expireAt := time.Now().Add(60 * time.Second)
		key, _ := db.Key().Get("name")
		testx.AssertEqual(t, *key.ETime/1000, expireAt.UnixMilli()/1000)
	})

	t.Run("update pexpire", func(t *testing.T) {
		db, red := getDB(t)
		defer db.Close()
		_ = db.Str().SetExpires("name", "alice", 60*time.Second)

		cmd := redis.MustParse(parse, "pexpire name 30000")
		conn := redis.NewFakeConn()
		res, err := cmd.Run(conn, red)
		testx.AssertNoErr(t, err)
		testx.AssertEqual(t, res, true)
		testx.AssertEqual(t, conn.Out(), "1")

		expireAt := time.Now().Add(30 * time.Second)
		key, _ := db.Key().Get("name")
		testx.AssertEqual(t, *key.ETime/1000, expireAt.UnixMilli()/1000)
	})

	t.Run("set to zero", func(t *testing.T) {
		db, red := getDB(t)
		defer db.Close()
		_ = db.Str().Set("name", "alice")

		cmd := redis.MustParse(parse, "pexpire name 0")
		conn := redis.NewFakeConn()
		res, err := cmd.Run(conn, red)
		testx.AssertNoErr(t, err)
		testx.AssertEqual(t, res, true)
		testx.AssertEqual(t, conn.Out(), "1")

		key, _ := db.Key().Get("name")
		testx.AssertEqual(t, key.Exists(), false)
	})

	t.Run("negative", func(t *testing.T) {
		db, red := getDB(t)
		defer db.Close()
		_ = db.Str().Set("name", "alice")

		cmd := redis.MustParse(parse, "pexpire name -1000")
		conn := redis.NewFakeConn()
		res, err := cmd.Run(conn, red)
		testx.AssertNoErr(t, err)
		testx.AssertEqual(t, res, true)
		testx.AssertEqual(t, conn.Out(), "1")

		key, _ := db.Key().Get("name")
		testx.AssertEqual(t, key.Exists(), false)
	})

	t.Run("not found", func(t *testing.T) {
		db, red := getDB(t)
		defer db.Close()
		_ = db.Str().Set("name", "alice")

		cmd := redis.MustParse(parse, "pexpire age 1000")
		conn := redis.NewFakeConn()
		res, err := cmd.Run(conn, red)
		testx.AssertNoErr(t, err)
		testx.AssertEqual(t, res, false)
		testx.AssertEqual(t, conn.Out(), "0")

		key, _ := db.Key().Get("age")
		testx.AssertEqual(t, key.Exists(), false)
	})
}
