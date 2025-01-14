# TRADING SERVICE

## Redis server
```json
{
  "address": "localhost:6379",
  "password": "",
  "DB": 0
}
```

## GoCryptoTrader Configuration
Please make sure to make configuration file like `config_example.json` and save on `$HOME/.gocryptotrader/config.json` for linux or `AppData/Roaming/GoCryptoTrader/config.json` on windows

## Exchange Platform

| Name | Avaliable Assets | Connection Type|
|------|-----------------|----------------|
| Binance | Spot, UsdtMarginedFutures, CoinMArginedFutures | REST, Websocket |
| Okx | Spot, PerpetualSwap | REST |
| Deribit | Spot, Futures | REST , Websocket |
| Bitmex | Spot, PerpetualContract | REST, Websocket |
| BTSE | Spot | REST, WebSocket |

## Request

### Type

1. Currency pair
   ```json
   {
    "base": string,
    "delimiter": string,
    "quote": string,
   }
   ```

### Submit Order

- Saved on redis as list with proto binary value
- Redis key: **`submitOrder`**
- Request: 
```json
{
  "exchange": string,
  "pair": string,
  "side": "buy" | "sell",
  "order_type": "market" | "limit",
  "asset_type": string, // refer to exchange enable asset
  "amount": number,
  "price": number,
  "client_order_id": string,
}
```

### Modify Order

- saved on redis as list with proto binary value
- Redis key: `modifyOrder`
- request:
  ```json
  {
    "exchange": string,
    "pair": string,
    "side": "buy" | "sell",
    "order_type": "market" | "limit",
    "asset_type": string, // refer to exchange enable asset
    "amount": number,
    "price": number,
    "client_order_id": string,
    "order_id": string,
  }
  ```

### Cancel Order

- saaved on redis as list with proto binary value
- redis key: `cancelOrder`
- request:
  ```json
  {
    "exchange": string,
    "client_order_id": string,
    "order_id": string,
    "pair": string,
    "asset_type": string,
    "side": "BUY" | "SELL",
    "order_type": "Limit" | "Market",
  }
  ```

### Execution Report

- saved on redis as list with proto binary value
- redis key: `executionReport`
- defitinition:
  ```json
  {
    "exchange": string,
    "client_order_id": string,
    "id": string, //refer as order_id in other definition
    "base_currency": string,
    "quote_currency": string,
    "asset_type": string,
    "order_side": string,
    "order_type": string,
    "creation_time": string, // date time string
    "update_time": string,  // date time string
    "status": string,
    "price": number,
    "average_price": number,
    "amont": number,
    "filled_amount": number,
    "remaining_amount": number,
    "fee": number,
    "cost": number,
    "trades": [
      {
        "id": string,
        "creation_time": string, // date time string
        "price": number,
        "amount": number,
        "exchange": string,
        "asset_type": string,
        "order_side": string,
        "fee": number,
        "total": number
      }
    ]
  }