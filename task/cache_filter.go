package task

import (
	"github.com/bits-and-blooms/bloom/v3"
	lru "github.com/hashicorp/golang-lru"
	"github.com/sirupsen/logrus"
)

type AdvanceQueryFunc = func(key string) (interface{}, bool, error)

type CacheFilter struct {
	cache        *lru.Cache
	filter       *bloom.BloomFilter
	findInDBFunc AdvanceQueryFunc
}

func NewCacheFilter(cacheSize int, bloomBits uint, bloomHashing uint, queryFunc AdvanceQueryFunc) (c *CacheFilter, err error) {
	//result.bloomFilter = bloom.New(20*1024*1024*8, 20)
	c = &CacheFilter{
		filter:       bloom.New(bloomBits, bloomHashing),
		findInDBFunc: queryFunc,
	}

	c.cache, err = lru.New(cacheSize)
	if err != nil {
		return
	}

	return
}

func (c *CacheFilter) Get(key string) (value interface{}, ok bool) {
	value, ok = c.cache.Get(key)
	if ok {
		return
	}

	//warning!!! filter may not from a full collection.  if filter tests not exist, it may exist in db
	if !c.filter.TestString(key) {
		return
	}
	value, ok, err := c.findInDBFunc(key)
	if err != nil {
		logrus.Warnf("query cache info in db error:%v", err)
		return
	}

	if ok {
		c.cache.Add(key, value)
	}

	return
}

func (c *CacheFilter) Add(key string, value interface{}) {
	c.cache.Add(key, value)
	c.filter.AddString(key)
}

func (c *CacheFilter) clean() {
	c.cache.Purge()
	c.filter.ClearAll()
}
