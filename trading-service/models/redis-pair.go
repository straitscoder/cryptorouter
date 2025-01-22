package model

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/redis/go-redis/v9"
)

const (
	pairKey     = "pair"
	baseListKey = "baseList"
)

type Pair struct {
	Base      string `json:"base"`
	Quote     string `json:"quote"`
	Delimiter string `json:"delimiter"`
}

func AddPair(ctx context.Context, pair Pair) error {
	jsonPair, err := json.Marshal(pair)
	if err != nil {
		return err
	}

	pairHash, err := rdClient.HGetAll(ctx, pairKey).Result()
	if err != nil {
		if err == redis.Nil {
			pairHash = make(map[string]string)
			pairHash[pair.Base] = string(jsonPair)
			if err := rdClient.RPush(ctx, baseListKey, pair.Base).Err(); err != nil {
				return err
			}
			if err := rdClient.HSet(ctx, pairKey, pairHash).Err(); err != nil {
				return err
			}

			return err
		}
	}

	if err := rdClient.RPush(ctx, baseListKey, pair.Base).Err(); err != nil {
		return err
	}
	if err := rdClient.HSetNX(ctx, pairKey, pair.Base, jsonPair).Err(); err != nil {
		return err
	}
	return nil
}

func CheckExistingandAddPair(ctx context.Context, base string) error {
	jsonPair, err := rdClient.HGet(ctx, pairKey, base).Bytes()
	if err != nil {
		if err == redis.Nil {
			pair := Pair{
				Base:      base,
				Delimiter: "-",
				Quote:     "USDT",
			}
			if err := AddPair(ctx, pair); err != nil {
				return err
			}

			return nil
		}

		return err
	}

	var pair Pair
	err = json.Unmarshal(jsonPair, &pair)
	if err != nil {
		return err
	}

	if pair.Base != "" {
		return nil
	}
	return errors.New("non existing pair")
}

func GetPairs(ctx context.Context) ([]Pair, error) {
	var pairs []Pair
	baseList, err := rdClient.LRange(ctx, baseListKey, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return pairs, nil
		}
		return pairs, err
	}

	pairs = make([]Pair, len(baseList))
	for i := range baseList {
		jsonPair, err := rdClient.HGet(ctx, pairKey, baseList[i]).Bytes()
		if err != nil {
			if err == redis.Nil {
				return pairs, nil
			}
			return pairs, err
		}

		var pair Pair
		err = json.Unmarshal(jsonPair, &pair)
		if err != nil {
			return pairs, err
		}
		pairs = append(pairs, pair)
	}

	return pairs, nil
}
