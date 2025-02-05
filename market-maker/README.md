# MARKET MAKER

## Feature
1. Fetch ticker from several exchange based on CFT Fix server serurity details and calculate fair price based on exchange trading volumes
2. Contantly place and orders on CFT fix server to build plarform fair price
3. Automatically place a hedging order on available exchange that provide the fair price

## Requirement
1. config.json file from GoCryptoTrader template that saved on `~/.gocryptotrader` for Linux or `~/AppData/Roaming/GoCryptoTrader` for Windows
2. Designated exchange account and API key
3. Redis server on `localhost:6379`
4. `price.cfg` on working directory to config price, quantity, gap tolerance, and order level depth
5. `market-maker.cfg` on working directory basic fix protocol configuration

## Application Process
1. Start: GoCryptoTrader Exchange Manager, Websocket Routine, FIX Initiator
2. Cancel all existing old order in redis database
3. Fetch and calculate fair price
4. Place or modify to fit fair price and price configuration file
5. Automatically create a hedging order if order that placed in CFT Fix server has been filled
6. if interupt signal has been captured will cancelled all exisiting order
   > every cancelled order will be erase from redis

## Feature Flow

### Fetch and calculate fair price
For this feature we use memory store, mutex, and 2 type of connection request:
- REST API
  1. Every 500 milisecond market maker will make a request for available currency in CCX on every available exchange to get its current ticker
  2. calculate its ticker by summarized its bid and ask price then divided by 2
  3. look at memory store for currency fair price
  4. if not exist will saved to memory store
  5. if the fair price for this currency exist, will check if the current fair price volume is greater existing fairprice will be replaced
- Websocket
  1. Market maker will subscribe for ticker channel on every available exchange
  2. When receive ticker data from these channel it will check if the currency available on CCX or not
  3. When the currency is available it calculate fairprice with the same method as REST
  4. Check on memory store wheter the current currency exist or not
  5. When not exist will save the fairprice to memory store
  6. When exist will check if the current fairprice volume is greater then will be replaced the existing

### Place Multiple Order
> this process will be done every 500 milisecond
1. Iterate through available fairprices on memory store
2. Check for active order on redis database
3. When not orders exist will calculate orders' price based on current fair price, price configuration and price multiplier from CCX
4. Place orders on both side based on order level configuration
5. When exist will check the price difference on fairprice and existing order
6. When the price difference exceed the gap tolerance configuration, market maker will calculate and modify existing orders to current fairprice

### Hedging
> this feature will be trigered if receive filled or partial filled execution report from CCX
1. Change order side to the opposite
2. Change order type to MARKET
3. Saved order detail on redis' `CounterOrderQueue`
4. Every 500 milisecond will check on redis `CounterOrderQueue`
5. When exist market maker will place an order on exchange that provide fairprice

