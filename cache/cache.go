package cache

import (
	"os"
	"path"
	"time"

	"go.etcd.io/bbolt"
)

const (
	fileName         = ".tiktok_cache.db"
	bucketName       = "tiktok"
	timestampKeyName = "timestamp"
)

type Cache struct {
	db *bbolt.DB
}

func New(cacheDir string) *Cache {
	if cacheDir == "" {
		var err error

		cacheDir, err = os.UserCacheDir()

		if err != nil {
			cacheDir, err = os.Getwd()
			if err != nil {
				panic(err)
			}
		}
	}

	db, err := bbolt.Open(path.Join(cacheDir, fileName), 0600, nil)

	if err != nil {
		panic(err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})

	if err != nil {
		panic(err)
	}

	return &Cache{db}
}

func (c *Cache) Set(key []byte, value []byte) error {

	return c.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		return bucket.Put(key, value)
	})
}

func (c *Cache) Get(key []byte) []byte {

	var value []byte
	c.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		value = bucket.Get(key)
		return nil
	})

	return value
}

func (c *Cache) WriteTimestamp() {
	stamp, _ := time.Now().MarshalBinary()
	c.Set([]byte(timestampKeyName), stamp)
}

func (c *Cache) GetTimestamp() time.Time {
	value := c.Get([]byte(timestampKeyName))

	if value != nil {
		var stamp time.Time
		err := stamp.UnmarshalBinary(value)

		if err == nil {
			return stamp
		}
	}
	return time.Time{}
}
