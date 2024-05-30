package core

import (
	"fmt"
	"github.com/go-redis/redis"
	"sync"
	"time"
)

type (
	RedisConnect struct {
		lock       sync.Mutex
		cli        *redis.Client
		conf       RedisConf
		errChannel chan error
	}

	// RedisConf is a config proxy to redis.Options
	RedisConf struct {
		Host     string `json:"host" yaml:"host"`
		Port     string `json:"port" yaml:"port"`
		Pass     string `json:"pass" yaml:"pass"`
		Database int    `json:"database" yaml:"database"`

		// Maximum number of retries before giving up.
		// Default is to not retry failed commands.
		MaxRetries int `json:"max_retries" yaml:"max_retries"`

		// Minimum backoff between each retry.
		// Default is 8 milliseconds; -1 disables backoff.
		MinRetryBackoff int64 `json:"min_retry_backoff_ms" yaml:"min_retry_backoff_ms"`

		// Maximum backoff between each retry.
		// Default is 512 milliseconds; -1 disables backoff.
		MaxRetryBackoff int64 `json:"max_retry_backoff_ms" yaml:"max_retry_backoff_ms"`

		// Dial timeout for establishing new connections.
		// Default is 5 seconds.
		DialTimeout int64 `json:"dial_timeout_s" yaml:"dial_timeout_s"`

		// Timeout for socket reads. If reached, commands will fail
		// with a timeout instead of blocking. Use value -1 for no timeout and 0 for default.
		// Default is 3 seconds.
		ReadTimeout int64 `json:"read_timeout_s" yaml:"read_timeout_s"`

		// Timeout for socket writes. If reached, commands will fail
		// with a timeout instead of blocking.
		// Default is ReadTimeout.
		WriteTimeout int64 `json:"write_timeout_s" yaml:"write_timeout_s"`

		// Maximum number of socket connections.
		// Default is 10 connections per every CPU as reported by runtime.NumCPU.
		PoolSize int `json:"pool_size" yaml:"pool_size"`

		// Minimum number of idle connections which is useful when establishing
		// new connection is slow.
		MinIdleConns int `json:"min_idle_conns" yaml:"min_idle_conns"`

		// Connection age at which client retires (closes) the connection.
		// Default is to not close aged connections.
		MaxConnAge int64 `json:"max_conn_age_s" yaml:"max_conn_age_s"`

		// Amount of time client waits for connection if all connections
		// are busy before returning an error.
		// Default is ReadTimeout + 1 second.
		PoolTimeout int64 `json:"pool_timeout_s" yaml:"pool_timeout_s"`

		// Amount of time after which client closes idle connections.
		// Should be less than server's timeout.
		// Default is 5 minutes. -1 disables idle timeout check.
		IdleTimeout int64 `json:"idle_timeout_s" yaml:"idle_timeout_s"`

		// Frequency of idle checks made by idle connections reaper.
		// Default is 1 minute. -1 disables idle connections reaper,
		// but idle connections are still discarded by the client
		// if IdleTimeout is set.
		IdleCheckFrequency int64 `json:"idle_check_frequency_s" yaml:"idle_check_frequency_s"`
	}
)

func NewRedisConnect(conf RedisConf) *RedisConnect {
	c := &RedisConnect{
		conf: conf,
	}
	c.GetConnection()

	return c
}

func (c *RedisConnect) GetConnection() *redis.Client {
	var err error

	c.lock.Lock()
	if c.cli != nil {
		_, err := c.cli.Ping().Result()
		if err != nil {
			cErr := c.cli.Close()
			if cErr != nil {
				fmt.Printf("[REDIS] close after PING err: %s", err)
			}
			c.cli = nil

		} else {
			c.lock.Unlock()
			return c.cli
		}
	}

	for {
		c.cli = redis.NewClient(c.conf.ResolveOptions())

		if err = c.cli.Ping().Err(); err != nil {
			err = c.cli.Close()
			if err != nil {
				fmt.Printf("[REDIS] close after PING after open connection err {%s}", err)
			}
			c.cli = nil
			time.Sleep(time.Second)

			continue
		}

		break
	}

	c.lock.Unlock()

	return c.cli
}

func (c *RedisConnect) Close() error {
	return c.cli.Close()
}

func (conf *RedisConf) ResolveOptions() *redis.Options {

	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%s", conf.Host, conf.Port),
		Password:     conf.Pass,
		DB:           conf.Database,
		MaxRetries:   conf.MaxRetries,
		Network:      "tcp",
		PoolSize:     conf.PoolSize,
		MinIdleConns: conf.MinIdleConns,
	}

	if conf.MinRetryBackoff >= 0 {
		opts.MinRetryBackoff = time.Millisecond * time.Duration(conf.MinRetryBackoff)
	} else if conf.MinRetryBackoff == -1 {
		opts.MinRetryBackoff = -1
	}

	if conf.MaxRetryBackoff >= 0 {
		opts.MaxRetryBackoff = time.Millisecond * time.Duration(conf.MaxRetryBackoff)
	} else if conf.MaxRetryBackoff == -1 {
		opts.MaxRetryBackoff = -1
	}

	if conf.DialTimeout > 0 {
		opts.DialTimeout = time.Second * time.Duration(conf.DialTimeout)
	}

	if conf.ReadTimeout >= 0 {
		opts.ReadTimeout = time.Second * time.Duration(conf.ReadTimeout)
	} else if conf.ReadTimeout == -1 {
		opts.ReadTimeout = -1
	}

	if conf.WriteTimeout >= 0 {
		opts.WriteTimeout = time.Second * time.Duration(conf.WriteTimeout)
	} else if conf.WriteTimeout == -1 {
		opts.WriteTimeout = -1
	}

	if conf.PoolTimeout > 0 {
		opts.PoolTimeout = time.Second * time.Duration(conf.PoolTimeout)
	}

	if conf.IdleTimeout > 0 {
		opts.IdleTimeout = time.Second * time.Duration(conf.IdleTimeout)
	}

	if conf.IdleCheckFrequency > 0 {
		opts.IdleCheckFrequency = time.Second * time.Duration(conf.IdleCheckFrequency)
	}

	if conf.MaxConnAge > 0 {
		opts.MaxConnAge = time.Second * time.Duration(conf.MaxConnAge)
	}

	return opts
}
