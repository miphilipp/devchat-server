package session

import (
	"strings"
	"strconv"
	"time"
	"fmt"
	//"errors"

	"github.com/go-redis/redis"
)


type SessionPersistance interface {
	BlackList(username string, token string, exp float64) error
	IsBlackListed(username string, token string) (bool, error)
	Store(username string, token string, exp int64) error
	BlackListAll(username string) error
}

type inMemorySessionPersistance struct {
	RedisClient *redis.Client
}

func NewInMemorySessionPersistance(addr string, password string) (*inMemorySessionPersistance, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,  // use default DB
	})

	_, err := redisClient.Ping().Result()
	if err != nil {
		return nil, err
	}

	return &inMemorySessionPersistance {
		RedisClient: redisClient,
	}, nil
}

func (p inMemorySessionPersistance) IsBlackListed(username string, token string) (bool, error) {
	list, err := p.RedisClient.LRange("invalid__" + username, 0, -1).Result()
	if err != nil {
		return false, err
	}

	isBlackListed := false
	for _, pair := range list {
		elements := strings.Split(pair, ",")
		if len(elements) != 2 {
			panic(fmt.Errorf("%s", "Redis format error"))
			continue
		}

		if elements[0] == token {
			isBlackListed = true
		}

		unixTimeSeconds, err := strconv.ParseInt(elements[1], 10, 64)
		if err != nil {
			return false, err
		}

		if time.Unix(unixTimeSeconds, 0).Before(time.Now().UTC()) {
			err = p.RedisClient.LRem("invalid__" + username, 0, pair).Err()
			if err != nil {
				return false, err
			}
		}
	}

	return isBlackListed, nil
}

func (p inMemorySessionPersistance) BlackList(username string, token string, exp float64) error {
	values := []string{token, strconv.FormatFloat(exp, 'f', 0, 64)}
	searchPair := strings.Join(values, ",")

	list, err := p.RedisClient.LRange("valid__" + username, 0, -1).Result()
	if err != nil {
		return err
	}

	for _, pair := range list {
		elements := strings.Split(pair, ",")
		if len(elements) != 2 {
			panic(fmt.Errorf("%s", "Redis format error"))
			continue
		}

		unixTimeSeconds, err := strconv.ParseInt(elements[1], 10, 64)
		if err != nil {
			return err
		}

		// Wenn gefunden und noch nicht abgelaufen, dann einfügen
		if searchPair == pair && time.Unix(unixTimeSeconds, 0).After(time.Now().UTC()) {
			err = p.RedisClient.RPush("invalid__" + username, pair).Err()
			if err != nil {
				return err
			}
		}
		
		// Wenn gefunden oder abgelaufen, dann entfernen
		if searchPair == pair || time.Unix(unixTimeSeconds, 0).Before(time.Now().UTC()) {
			err = p.RedisClient.LRem("valid__" + username, 0, pair).Err()
			if err != nil {
				return err
			}
		}
		
	}
	return nil
}

func (p inMemorySessionPersistance) Store(username string, token string, exp int64) error {
	values := []string{token, strconv.FormatInt(exp, 10)}
	return p.RedisClient.RPush("valid__" + username, strings.Join(values, ",")).Err()
}

func (p inMemorySessionPersistance) BlackListAll(username string) error{
	list, err := p.RedisClient.LRange("valid__" + username, 0, -1).Result()
	if err != nil {
		return err
	}

	for _, pair := range list {
		elements := strings.Split(pair, ",")
		if len(elements) != 2 {
			continue
		}

		unixTimeSeconds, err := strconv.ParseInt(elements[1], 10, 64)
		if err != nil {
			return err
		}

		err = p.RedisClient.LRem("valid__" + username, 0, pair).Err()
		if err != nil {
			return err
		}

		if time.Unix(unixTimeSeconds, 0).After(time.Now().UTC()) {
			err = p.RedisClient.RPush("invalid__" + username, pair).Err()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p inMemorySessionPersistance) Persist() error  {
	return p.RedisClient.BgSave().Err()
}