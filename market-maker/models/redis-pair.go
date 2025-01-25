package model

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const (
	pairKey     = "pair"
	baseListKey = "baseList"
)

type Pair struct {
	Base               string  `json:"base"`
	Quote              string  `json:"quote"`
	Delimiter          string  `json:"delimiter"`
	ContractMultiplier float64 `json:"contractMultiplier"`
	PriceIncrement     float64 `json:"priceIncrement"`
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

func CheckExistingandAddPair(ctx context.Context, base string, contractMultiplier string, priceIncrement string) error {
	jsonPair, err := rdClient.HGet(ctx, pairKey, base).Bytes()
	if err != nil {
		if err == redis.Nil {
			cMultiplier, err := strconv.ParseFloat(contractMultiplier, 64)
			if err != nil {
				return err
			}

			prcIncrement, err := strconv.ParseFloat(priceIncrement, 64)
			if err != nil {
				return err
			}

			pair := Pair{
				Base:               base,
				Delimiter:          "-",
				Quote:              "USDT",
				ContractMultiplier: cMultiplier,
				PriceIncrement:     prcIncrement,
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

func GetPair(ctx context.Context, base string) (*Pair, error) {
	var pair Pair
	jsonPair, err := rdClient.HGet(ctx, pairKey, base).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	err = json.Unmarshal(jsonPair, &pair)
	if err != nil {
		return nil, err
	}
	switch pair.Base {
	// only available for this base
	case "AVAX", "BCH", "BTC", "BNB", "ETH", "LTC", "SOL":
		return &pair, nil
	default:
		return nil, nil
	}
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
