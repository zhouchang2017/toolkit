package redisconfig

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/zhouchang2017/toolkit/log"
)

var (
	instance = map[string]*redis.Client{}
)

type Configs map[string]*Config

type Config struct {
	Addr     string
	Password string
	DB       int
}

func (c *Config) NewClient() *redis.Client {
	if c.Addr == "" {
		c.Addr = "localhost:6379"
	}
	opt := &redis.Options{
		Addr:     c.Addr,
		Password: c.Password,
		DB:       c.DB,
	}
	return redis.NewClient(opt)
}

// on server starting
func (c *Configs) Init() (err error) {
	if c == nil {
		return nil
	}
	for name, config := range *c {
		instance[name] = config.NewClient()
	}
	return nil
}

// on server closing
func (c *Configs) Close() {
	for name, client := range instance {
		if err := client.Close(); err != nil {
			log.Logger.Errorf("redis [%s] close err:%s", name, err.Error())
		}
	}
}

// global api
func Get(name string) (*redis.Client, error) {
	if client, ok := instance[name]; ok {
		return client, nil
	}
	return nil, fmt.Errorf("%s redis not found", name)
}
