package task

import (
	"testing"
	"time"

	"github.com/chainmonitor/output/mysqldb"
)

var dbtest *mysqldb.MysqlDB

func init() {
	host := `root:123456sj@tcp(127.0.0.1:3306)/chain_bsc?charset=utf8mb4`
	host = `test:123@tcp(127.0.0.1:3306)/chain?charset=utf8mb4`
	dbtest, _ = mysqldb.NewMysqlDb(host, 5)

}

func TestBalance(t *testing.T) {
	addr := "0x57a58f6caa47c340e24dd5633b335a12bec7d402"
	latestHeight := 9999999999
	cache, err := NewCacheFilter(1, 1024*1024*8*20, 20, func(key string) (interface{}, bool, error) {
		res, err := dbtest.GetBalance(key)
		if err != nil {
			return nil, false, err
		}

		if res == nil {
			return nil, false, nil
		}

		return res.Height, true, nil
	})
	if err != nil {
		t.Fatalf("new cache err %v", err)
	}
	_, ok := cache.Get(addr)
	if ok {
		t.Fatal("return true when not added")
	}
	cache.Add(addr, latestHeight)
	height, ok := cache.Get(addr)
	if !ok {
		t.Fatal("return false after added")
	}
	if height != latestHeight {
		t.Fatalf("The returned value is incorrect")
	}
	cache.Add("xxxx", 1)         //这里lruCacheSize是1，此处模拟lruCache失效
	height, ok = cache.Get(addr) //此时height来自数据库
	if !ok {
		t.Fatal("return false after added")
	}
	t.Logf("the height from db is %v", height)
}

func TestBalanceErc20(t *testing.T) {
	addr := "0x57a58f6caa47c340e24dd5633b335a12bec7d402"
	contractAddr := "0x04f535663110a392a6504839beed34e019fdb4e0"
	latestHeight := 9999999999
	cache, err := NewCacheFilter(1, 1024*1024*8*20, 20, func(key string) (interface{}, bool, error) {
		res, err := dbtest.GetBalanceErc20(addr, contractAddr)
		if err != nil {
			return nil, false, err
		}

		if res == nil {
			return nil, false, nil
		}

		return res.Height, true, nil
	})
	if err != nil {
		t.Fatalf("new cache err %v", err)
	}
	key := addr + "|" + contractAddr
	_, ok := cache.Get(key)
	if ok {
		t.Fatal("return true when not added")
	}
	cache.Add(key, latestHeight)
	height, ok := cache.Get(key)
	if !ok {
		t.Fatal("return false after added")
	}
	if height != latestHeight {
		t.Fatalf("The returned value is incorrect")
	}
	cache.Add("xxxx", 1)        //这里lruCacheSize是1，此处模拟lruCache失效
	height, ok = cache.Get(key) //此时height来自数据库
	if !ok {
		t.Fatal("return false after added")
	}
	t.Logf("the height from db is %v", height)
}

func TestCacheClean(t *testing.T) {
	cache, err := NewCacheFilter(1, 1024*1024*8*20, 20, func(key string) (interface{}, bool, error) {
		return 1, true, nil
	})
	if err != nil {
		t.Fatal("init cache failed")
	}
	start := time.Now()
	cache.clean()
	end := time.Now()
	t.Logf("start:%v, enf %v", start, end)
}
