package redis

import (
	"fmt"
	"time"
	"strings"
	"sync"

	"github.com/abdfnx/redui/core"
	"github.com/abdfnx/redui/config"

	"github.com/gdamore/tcell"
	goRedis "github.com/go-redis/redis/v8"
)

// RedisClient is a redis client which wraps single or cluster client
type RedisClient interface {
	Keys(pattern string) *goRedis.StringSliceCmd
	Scan(cursor uint64, match string, count int64) *goRedis.ScanCmd
	Type(key string) *goRedis.StatusCmd
	TTL(key string) *goRedis.DurationCmd
	Get(key string) *goRedis.StringCmd
	LRange(key string, start, stop int64) *goRedis.StringSliceCmd
	SMembers(key string) *goRedis.StringSliceCmd
	ZRangeWithScores(key string, start, stop int64) *goRedis.ZSliceCmd
	HKeys(key string) *goRedis.StringSliceCmd
	HGet(key, field string) *goRedis.StringCmd
	Process(cmd goRedis.Cmder) error
	Do(args ...interface{}) *goRedis.Cmd
	Info(section ...string) *goRedis.StringCmd
}

func NewRedisClient(conf config.Config, outputChan chan core.OutputMessage) RedisClient {
	if conf.Cluster {
		options := &goRedis.ClusterOptions{
			Addrs:    []string{fmt.Sprintf("%s:%d", conf.Host, conf.Port)},
			Password: conf.Password,
		}

		return goRedis.NewClusterClient(options)
	}

	options := &goRedis.Options{
		Addr:         fmt.Sprintf("%s:%d", conf.Host, conf.Port),
		DB:           conf.DB,
		Password:     conf.Password,
		WriteTimeout: 3 * time.Second,
		ReadTimeout:  2 * time.Second,
	}

	client := goRedis.NewClient(options)
	if conf.Debug {
		client.WrapProcess(func(oldProcess func(cmd goRedis.Cmder) error) func(cmd goRedis.Cmder) error {
			return func(cmd goRedis.Cmder) error {

				outputChan <- core.OutputMessage{Color: tcell.ColorOrange, Message: fmt.Sprintf("redis: <%s>", cmd)}
				err := oldProcess(cmd)

				return err
			}
		})
	}

	return client
}

func RedisExecute(client RedisClient, command string) (interface{}, error) {
	stringArgs := strings.Split(command, " ")
	var args = make([]interface{}, len(stringArgs))

	for i, s := range stringArgs {
		args[i] = s
	}

	return client.Do(args...).Result()
}

var redisKeys = make([]string, 0)

func KeysWithLimit(client RedisClient, key string, maxScanCount int) (redisKeys []string, err error) {
	var cursor uint64 = 0
	var keys []string
	var scanCount = 0

	for scanCount < maxScanCount || maxScanCount == -1{
		scanCount++

		keys, cursor, err = client.Scan(cursor, key, 100).Result()

		if err != nil {
			return
		}

		redisKeys = append(redisKeys, keys...)

		if cursor == 0 {
			break
		}
	}

	return
}
