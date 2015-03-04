// Copyright 2013 Beego Authors
// Copyright 2014 Unknwon
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package redistore

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"time"
	"unsafe"

	"github.com/garyburd/redigo/redis"
	"github.com/lunny/log"
	"github.com/lunny/tango"

	"github.com/tango-contrib/session"
)

var _ session.Store = &RedisStore{}

type Options struct {
	Host     string
	Port     string
	Password string
	DbIndex  int
	MaxAge   time.Duration
}

// RedisStore represents a redis session store implementation.
type RedisStore struct {
	Options
	Logger tango.Logger
	pool   *redis.Pool
}

func preOptions(opts []Options) Options {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Host == "" {
		opt.Host = "127.0.0.1"
	}
	if opt.Port == "" {
		opt.Port = "6379"
	}
	if opt.MaxAge == 0 {
		opt.MaxAge = session.DefaultMaxAge
	}
	return opt
}

// NewRedisStore creates and returns a redis session store.
func New(opts ...Options) *RedisStore {
	opt := preOptions(opts)
	var pool = &redis.Pool{
		MaxIdle:     5,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			// the redis protocol should probably be made sett-able
			c, err := redis.Dial("tcp", opt.Host+":"+opt.Port)
			if err != nil {
				return nil, err
			}
			if len(opt.Password) > 0 {
				if _, err := c.Do("AUTH", opt.Password); err != nil {
					c.Close()
					return nil, err
				}
			} else {
				// check with PING
				if _, err := c.Do("PING"); err != nil {
					c.Close()
					return nil, err
				}
			}
			_, err = c.Do("SELECT", opt.DbIndex)
			return c, err
		},
		// custom connection test method
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if _, err := c.Do("PING"); err != nil {
				return err
			}
			return nil
		},
	}

	return &RedisStore{
		Options: opt,
		pool:    pool,
		Logger:  log.Std,
	}
}

func (c *RedisStore) serialize(value interface{}) ([]byte, error) {
	err := c.registerGobConcreteType(value)
	if err != nil {
		return nil, err
	}

	if reflect.TypeOf(value).Kind() == reflect.Struct {
		return nil, fmt.Errorf("serialize func only take pointer of a struct")
	}

	var b bytes.Buffer
	encoder := gob.NewEncoder(&b)

	err = encoder.Encode(&value)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (c *RedisStore) deserialize(byt []byte) (ptr interface{}, err error) {
	b := bytes.NewBuffer(byt)
	decoder := gob.NewDecoder(b)

	var p interface{}
	err = decoder.Decode(&p)
	if err != nil {
		return
	}

	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Struct {
		var pp interface{} = &p
		datas := reflect.ValueOf(pp).Elem().InterfaceData()

		sp := reflect.NewAt(v.Type(),
			unsafe.Pointer(datas[1])).Interface()
		ptr = sp
	} else {
		ptr = p
	}
	return
}

func (c *RedisStore) registerGobConcreteType(value interface{}) error {
	t := reflect.TypeOf(value)

	switch t.Kind() {
	case reflect.Ptr:
		v := reflect.ValueOf(value)
		i := v.Elem().Interface()
		gob.Register(i)
	case reflect.Struct, reflect.Map, reflect.Slice:
		gob.Register(value)
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Bool, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		// do nothing since already registered known type
	default:
		return fmt.Errorf("unhandled type: %v", t)
	}
	return nil
}

// Set sets value to given key in session.
func (s *RedisStore) Set(id session.Id, key string, val interface{}) error {
	bs, err := s.serialize(val)
	if err != nil {
		return err
	}
	_, err = s.Do("HSET", id, key, bs)
	if err == nil {
		// when write data, reset maxage
		_, err = s.Do("EXPIRE", id, s.MaxAge)
	}
	return err
}

// Get gets value by given key in session.
func (s *RedisStore) Get(id session.Id, key string) interface{} {
	val, err := s.Do("HGET", id, key)
	if err != nil {
		s.Logger.Errorf("redis HGET failed: %s", err)
		return nil
	}

	// when read data, reset maxage
	s.Do("EXPIRE", id, s.MaxAge)

	item, err := redis.Bytes(val, err)
	if err != nil {
		s.Logger.Errorf("redis.Bytes failed: %s", err)
		return nil
	}

	value, err := s.deserialize(item)
	if err != nil {
		s.Logger.Errorf("redis HGET failed: %s", err)
		return nil
	}
	return value
}

// Delete delete a key from session.
func (s *RedisStore) Del(id session.Id, key string) bool {
	_, err := s.Do("HDEL", id, key)
	return err == nil
}

func (s *RedisStore) Clear(id session.Id) bool {
	_, err := s.Do("DEL", id)
	return err == nil
}

func (s *RedisStore) Add(id session.Id) bool {
	return true
}

func (s *RedisStore) Do(cmd string, args ...interface{}) (interface{}, error) {
	conn := s.pool.Get()
	defer conn.Close()
	return conn.Do(cmd, args...)
}

func (s *RedisStore) Exist(id session.Id) bool {
	has, err := s.Do("EXISTS", id)
	return err == nil && has.(bool)
}

func (s *RedisStore) SetMaxAge(maxAge time.Duration) {
	s.MaxAge = maxAge
}

func (s *RedisStore) SetIdMaxAge(id session.Id, maxAge time.Duration) {
	if s.Exist(id) {
		s.Do("EXPIRE", id, s.MaxAge)
	}
}

func (s *RedisStore) Ping() error {
	_, err := s.Do("Ping")
	return err
}

func (s *RedisStore) Run() error {
	return s.Ping()
}
