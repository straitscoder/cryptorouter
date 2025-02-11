namespace py Compass.Trading
namespace cpp Compass.Trading
namespace java com.compass.trading

enum MsgType {
	ORDER_SINGLE = 68,
	ORDER_LIST = 69,
	ORDER_CANCEL_REQUEST = 70,
	ORDER_CANCEL_REPLACE_REQUEST = 71,
	ORDER_STATUS_REQUEST = 72,
	ORDER_INFO_MODIFICATION_REQUEST = 73,
	EXECUTION_REPORT = 56,
	ORDER_CANCEL_REJECT = 57,
}

enum OrderType{
	MARKET = 49,
	LIMIT = 50,
	STOP = 51,
	STOP_LIMIT = 52,
	MARKET_ON_CLOSE = 53,
	LIMIT_ON_CLOSE = 59,
	MARKET_IF_TOUCHED = 67,
	LIMIT_IF_TOUCHED = 68,
	BLOCK_TRADE = 91,
	CROSS_TRADE = 92,
	OCO_MIT_STOP = 93,	
	OCO_LIMIT_STOP = 94,	
	OCO_LIMIT_STOPLIMIT = 95,
}

enum OrderTypeMask {
    LIMIT = 1,
    LIMIT_FAK = 2,
    LIMIT_FOK = 4,
    LIMIT_GTC = 8,
    LIMIT_GTD = 16,
    LIMIT_ON_CLOSE = 32,

    MARKET = 64,
    MARKET_FAK = 128,
    MARKET_FOK = 256,
    MARKET_ON_OPEN = 512,
    MARKET_ON_CLOSE = 1024,
    MARKET_TO_LIMIT = 2048,
    MARKET_IF_TOUCHED = 4096,

    STOP = 8192,
    STOP_GTC = 16384,
    STOP_GTD = 32768,

    STOP_LIMIT = 65536,
    STOP_LIMIT_GTC = 131072,
    STOP_LIMIT_GTD = 262144,
    LIMIT_IF_TOUCHED = 524288,
}

enum OrderStatus{
	NEW = 48,
	PARTIALLY_FILLED,
	FILLED,
	DONE_FOR_DAY,
	CANCELED,
	REPLACED,
	PENDING_CANCEL,
	STOPPED,
	REJECTED,
	SUSPENDED,
	PENDING_NEW = 65,
	CALCULATED = 66,
	PENDING_REPLACE = 69
}

enum OrderStatusGroup{
	WORKING = 1,
	COMPLETED,
	FILLED,
	REJECTED,
}

enum UserPrivilege{
	CANCEL_ORDER = 1,
	OPEN_POSITION = 2,
	CLOSE_POSITION = 4,
}

enum UserStatus{
	ENABLE = 0,
	ONLINE,
	BLOCKED,
	DISABLED,
	OFFLINE,
}

enum Side{
	BUY = 49,
	SELL = 50,
}

enum MarketStatus{
	PRE_OPEN = 48,
	OPENED,
	PRE_CLOSE,
	CLOSED,
	SUSPENDED,
}

enum TimeInForce{
    DAY = 48,
    GOOD_TILL_CANCEL = 49,
    AT_THE_OPENING = 50,
    IMMEDIATE_OR_CANCEL = 51,
    FILL_OR_KILL = 52,
    GOOD_TILL_CROSSING = 53,
    GOOD_TILL_DATE = 54,
    AT_THE_CLOSE = 55,
}

enum ExecType {
	NEW = 48,
	PARTIAL_FILL = 49,
	FILL = 50,
	DONE_FOR_DAY = 51,
	CANCELED = 52,
	REPLACE = 53,
	PENDING_CANCEL = 54,
	STOPPED = 55,
	REJECTED = 56,
	SUSPENDED = 57,
	PENDING_NEW = 65,
	CALCULATED = 66,
	EXPIRED = 67,
	RESTATED = 68,
	PENDING_REPLACE = 69,
	TRADE = 70,
	TRADE_CORRECT = 71,
	TRADE_CANCEL = 72,
	ORDER_STATUS = 73,
	TRIGGERED = 81,
	CANCEL_REJECT = 82,
	REPLACE_REJECT = 83,
}

enum SettlementMethod{
	CASH,
	DELIVERY,
}

enum MarginMethod{
	FIXED_AMOUNT,
	PERCENTEGE,
	SPAN,
}

enum OffsetMethod{
	FIFO = 1,
	LIFO = 2,
	MANUAL = 3,
}

enum ErrorCode{
	OK = 0,
	SYSTEM_ERROR = 48,
	FUNCTION_NOT_IMPLEMENTED,
	ROUTING_FAILURE,
	INVALID_MSG_SIZE,
	EXCHANGE_NOT_AVAILABLE,

	INVALID_EXCHANGE_ID,
	INVALID_SECURITY_ID,
	INVALID_ACCOUNT_ID,

	ORDER_NOT_FOUND,
	ACCOUNT_BLOCKED,

	LIMIT_BREACHED,
	LIMIT_NOT_ASSIGNED,
	RATING_NOT_ASSIGNED,
	CASH_LIMIT_BREACHED,
	CLIP_LIMIT_BREACHED,

	LIMIT_PRICE_TOO_HIGH,
	LIMIT_PRICE_TOO_LOW,
	STOP_PRICE_TOO_HIGH,
	STOP_PRICE_TOO_LOW,

	INVALID_ORDER_STATE,
	INVALID_LIMIT_PRICE,
	INVALID_ORDER_SIZE,
	INVALID_ORDER_TYPE,
	INVALID_CL_ORD_ID,
	INVALID_ORDER_ID,

	INCORRECT_ORDER_INFO,

	ORDER_IS_FILLED,
	ORDER_IS_CANCELED,
	ORDER_IS_REJECTED,
	ORDER_IN_PENDING_NEW,
	ORDER_IN_PENDING_CANCEL,
	ORDER_IN_PENDING_REPLACE,

	MARKET_CLOSED,
	INSTRUMENT_SUSPENDED,

	NOT_ENOUGH_POSITION_TO_CLOSE,

	INVALID_BAR_INTERVAL,
	BAR_NOT_AVAILABLE,
	USER_ACCOUNT_NOT_ASSOCIATE,
	INVALID_USER_ID,
	USER_BLOCKED,

	NOT_ALLOW_TRADE,
	NOT_ALLOW_OPEN_POSITION,

	LONG_LIMIT_BREACHED,
	SHORT_LIMIT_BREACHED,
	
	INVALID_PASSWORD,
	PASSWORD_EXPIRED,
	SESSION_NOT_FOUND,
	SESSION_EXPIRED,
	
	SEP_CANCEL_INCOMING_ORDER = 100,
	SEP_CANCEL_RESTING_ORDER = 101,
	CANCEL_BY_FAK_TIMELIFE = 102,
	CANCEL_BY_FOK_TIMELIFE = 103,
	CANCEL_BY_REQUEST = 104,
	CANCEL_BY_ADMIN = 105,
	EXCHANGE_CANCEL = 106,
	INTERNAL_CANCEL = 107,

	UNIQUE_KEY_VIOLATION = 200,
	PRIMARY_KEY_VIOLATION = 201
	FOREIGN_KEY_VIOLATION = 202,
	
	EXCEED_MAX_ATTEMPT = 300,
}

enum SecurityType {
	FUTURE = 1,
	CALL = 2,
	PUT = 3,
	FORWARD = 4,
	SWAP = 5,
	WARRANT = 6,
	PHYSICAL = 7,
	STRATEGY = 20,
}

enum MarketSessionType {
	CONTINOUS = 1,
	CLOSE = 2,
	BREAK = 3,
	AUCTION = 4,
	PREOPEN = 5,
}

enum AccountType{
	BROKER = 1,
    INVESTOR = 2,
    OMNIBUS = 3,
}

enum Comparator{
	NOT_EQUAL = 0,
	EQUAL = 1,
	LESS = 2,
	LESS_OR_EQUAL = 3,
	GREATOR = 4,
	GREATOR_OR_EQUAL = 5,
	LIKE = 6,
	IN_LIST = 7,
}

enum Gender{
	MALE = 1,
	FEMALE = 2,
}

struct Filter{
	1: string key,
	2: string value,
	3: Comparator comparator,
}

enum SortingMethod{
	ASCENDING = 1,
	DESCENDING = 2,
}

enum BarSource{
	LAST_PRICE = 1,
	MID_PRICE = 2,
	BID_PRICE = 3,
	ASK_PRICE = 4,
}

enum Platform{
    RMS = 1,
    WEB_TRADER = 2,
    SMART_TRADER = 4,
    MOBILE_TRADER = 8,
    MAKET_MAKER = 16,
    TRADING_API = 4096,
}

enum DataType{
	TEXT = 1,
	NUMBER = 2, 
	BOOLEAN = 3,
	ENUM = 4,
	LIST = 5,
	JSON = 6, 
	BLOB = 7,
}

struct SortingCriteria{
	1: string field,
	2: SortingMethod sortingMethod,
}

enum OrderInfoMask {
	UNSOLICITED = 1,
	OPEN_NEW_POSITION = 2,
	LIFO_POSITION_MANAGEMENT = 4,
	SYNTHETIC_ORDER = 8,
	SYNCHETIC_ORDER_TRIGGERED = 16, 

	//The bits for China markets
	HEDGING = 32,
	OPEN_POSITION = 64,
	CLOSE_POSITION = 128,
	CLOSE_TODAY_POSITION = 256,
	CLOSE_YESTERDAY_POSITION = 512,
	OPEN_TODAY_POSITION = 1024,

	SYNTHETIC_TIMEINFORCE = 2048,
	SYNTHETIC_ORDERTYPE = 4096,
	
	SPREAD_ROLL_OVER = 8192,
	OMS_NO_OVERRIDE_MASKS = 16384,

	ENTRY = 32768,
	EXIT = 65536,
	ALGO = 131072,

	STOP_LOSS = 262144,
	PROFIT_TAKING = 524288,
	
	SPOT_OPEN_TRADE = 1048576,
	SPOT_CLOSE_TRADE = 2097152,

	BLOCK_TRADE = 4194304,
	CROSS_TRADE = 8388608,
	
	MARKET_MAKING = 16777216,
}

struct SearchCriteria{
	1:  optional i32 count,
	2:  optional i32 id,
	3:  optional i32 tradeDate,
	4:  optional string code,
	5:  optional string name,
	6:  optional string path,
	7:  optional string user,
	8:  optional string group,
	9:  optional string parent,
	10: optional string account,
	11: optional string currency,
	12: optional string exchange,
	13: optional string product,
	14: optional string instrument,
	15: optional AccountType accountType,
	16: optional SecurityType securityType,
	17: optional i32 ledgerTypeID,
	19: optional i32 accountID,
	20: optional i32 groupID,
	21: optional i32 userID,
	24: optional i32 parentID,
	25: optional i32 productID,	
	26: optional i32 instrumentID,
	
	30: optional i32 orderID,
	31: optional i32 tradeID,
	32: optional string executionID,
	33: optional OrderStatusGroup orderStatusGroup,
	35: optional i32 beginPeriod,
	36: optional i32 endPeriod,
	37: optional string requester,
	
	40: optional i32 refID,
	41: optional i32 beginIndex,
	42: optional i32 endIndex,
	43: optional i32 maxItemNumber,
	
	22: list<Filter> filters,
	23: list<SortingCriteria> sortingCriterias,
}

struct SystemSetting{
	1: required string section,
	2: required string key,
	3: required string value,
	4: optional DataType type,
}

enum DocumentType{
	IMAGE = 1,
	PDF = 2, 
	WORD = 3, 
	HTML = 4,
}

struct Document{
	1: required string uuid,
	2: required string name, 
	3: required DocumentType type,
	4: optional string description,
}

struct Currency{
	1: optional i32 id,
	2: required string code,
	3: optional string name,
	4: optional double fxRate,
	5: optional byte decimalPlace,
	6: optional byte symbol,
}

struct HolidayCenter{
	1: required string code,
	2: optional string name,
	3: optional string city,
	4: optional string country,
}

struct Holiday{
	1: required i32 date,
	2: optional string name,
	3: required string holidayCenter,
}

struct Exchange{
    1: optional i32 id,
	2: required string code,
	3: optional string name,
	5: optional string exchAcro,
	4: list<OrderType> orderTypes,
	6: optional i32 orderTypeMask,
}

struct ProductGroup{
	1: optional i32 id,
	2: required string code,
	3: optional string name,
	4: optional string icon,
}

struct Product{
	1: optional i32 id,
	2: required string code,
	3: optional string name,
	4: required string exchange,
	5: required string currency,
	6: optional string productGroup,

	7: optional double lotSize,
	8: optional string deliveryUnit,
	9: optional byte qtyExponent,

	10: required byte decimalPlace,
	11: required double tickSize,
	12: required double tickValue,
	13: optional double priceRange,
	14: optional double referencePrice,

	15: optional i32 reportLimit,
	16: optional i32 positionLimit,
	
	20: optional string icon,

	25: optional double spotMargin,
	23: optional double spreadMargin,
	24: optional double outrightMargin,
	21: optional MarginMethod marginMethod,
	22: optional SettlementMethod settlementMethod,

	50: optional string exchAcro,
	51: optional string combCode,
	52: optional string prodCode,
	53: optional string prodType,
}

struct ProductAttribute{
	1: required string product,
	2: required string group,
	3: required string key,
	4: required string value,
	5: optional DataType type,
}

struct Market{
	1: required string code,
	2: optional string name,
	3: required string holidayCenter,
	4: optional string productGroup,
}

struct MarketSession{
	1: optional i32 id,
	2: required string market,
	3: required MarketSessionType sessionType,
	4: required i32 startTime,
	5: required i32 endTime,
	6: required string dayMasks,
}

struct InstrumentLeg {
	1: required i32  legId,
	2: required Side legSide,
	3: required i32  legRatioQty, //qty multiplier
	4: required i32  instrumentID,
	5: optional double priceFactor,
}

struct Instrument{
	1: optional i32 id,
	2: required string code,
	3: optional string name,
	4: required string product,
	5: required string exchange,
	6: required string currency,
	7: optional i32 maturityDay,
	8: optional i32 maturityMonthYear,
    9: required SecurityType securityType,

	10: required byte decimalPlace,
	11: required double tickSize,
	12: required double tickValue,
	13: optional double strikePrice,

	14: optional i32 firstTradingDate,
	15: optional i32 lastTradingDate,
	16: optional i32 deliveryDate,
	17: optional BarSource barSource,

	18: optional double delta,
	19: optional double volatility,
	20: optional double referencePrice,
	21: optional double settlementPrice,
	22: optional i32 settlementPriceDate,
	23: optional double prevSettlementPrice,
	24: optional i32 prevSettlementPriceDate,

	25: optional double minPrice,
	26: optional double maxPrice,
	27: optional bool suspended,
    28: optional string extSecurityID,
	29: optional double marginPerLot,

	30: list<InstrumentLeg> legs,
	31: optional byte qtyExponent,

	40: optional i64 updateTime,
	41: optional string icon,

	50: optional string exchAcro,
	51: optional string combCode,
	52: optional string prodCode,
	53: optional string prodType,
	54: optional i32 period,
	55: optional i32 optPeriod,
	56: optional i32 optStrikePrice,
}

struct InstrumentAttribute{
	1: required string instrument,
	2: required string group,
	3: required string key,
	4: required string value,
	5: optional DataType type,
}

struct DepthEntry{
	1: required double price,
	2: optional i64 qty,
}

struct Quote{
	1: optional i32 id,
	2: required string instrumentCode,
	3: optional string instrumentName,
	4: required byte decimalPlace,
	5: optional SecurityType securityType,
	6: optional byte qtyExponent,

	10: optional i64 updateTime,
	11: optional i64 lastVolume,
	12: optional i64 totalVolume,
	13: optional i64 openInterest,

	14: optional double lastPrice, 
	15: optional double sessionLow,
	16: optional double sessionHigh,
	17: optional double openingPrice,
	18: optional double closingPrice,
	19: optional double settlementPrice,

	20: optional list<DepthEntry> bids,
	21: optional list<DepthEntry> asks,

	22: optional MarketStatus marketStatus,
	23: optional double referencePrice,
	24: optional double prevSettlementPrice,
	25: optional double lowLimitPrice,
	26: optional double highLimitPrice,
	
	50: optional string exchAcro,
	51: optional string combCode,
	52: optional string prodCode,
	53: optional string prodType,
	54: optional i32 period,
	55: optional i32 optPeriod,
	56: optional i32 optStrikePrice,
}

struct Order{
	1: i32 id,
	2: string user,
	3: string account,
	4: string exchange,
	5: string instrument,
	6: Side side,
	7: double price,
	8: double price2,
	9: double avgPrice,
	10: i32 qty,
	11: i32 cumQty,
	12: i32 leavesQty,
	13: i32 accountID,
	14: i32 strategyID,
	15: i32 instrumentID,
	16: string text,
	17: byte priceScale,
	18: OrderStatus status,	
	19: OrderType orderType,
	20: i32 rejectReason,
	21: i64 transactTime,
	22: i32 lastMsgID,
	23: double tickSize,
	24: double tickValue,
	25: ExecType lastExecType,
    26: list<string> reservedData,
    27: optional bool algorithmTrading,
    28: optional TimeInForce timeInForce,

	29: optional string accountPath,
	30: optional byte qtyExponent,
	
	31: optional double lastPx,
	32: optional i32 lastShares,
	
	33: optional i32 userID,
	34: optional i32 clOrdID,
	35: optional i64 orderInfoMask,
	
	36: optional byte msgType,
	37: optional byte cxlRejResponseTo,		
	
	38: optional double maxLoss,
	39: optional double maxProfit,

    //Audit
    40: optional string ipAddr,
    41: optional i32 port,
	42: optional i32 brokerID,
	43: optional i32 counterPartyID,
}

struct QuoteVersion{
	1: i32 instrumentID,
	2: i64 updateTime,
}

enum BarInterval{
	ONE_MINUTE = 60,
	FIVE_MINUTE = 300,
	FIFTEEN_MINUTE = 900,
	THIRTY_MINUTE = 1800,
	ONE_HOUR = 3600,
	ONE_DAY = 86400,
}

struct Bar{
    1: required double open,
    2: required double high,
    3: required double low,
    4: required double close,
    5: required i32 volume,
    6: required i64 time,
}

struct Bars{
    1: optional i32 requestID,
    2: required i32 instrumentID,
    3: optional string instrument,
    5: required BarInterval interval,
    4: required list<Bar> bars,
}

struct SettlementPrice{
	1: required i32 tradeDate,
	2: required i32 instrumentID,
	3: optional byte decimalPlace,
	4: required string instrumentCode,

	5: optional i64 updateTime,
	6: optional double minPrice,
	7: optional double maxPrice,
	8: optional double settlementPrice,
	9: optional double prevSettlementPrice,
}

enum LedgerGroup{
    INFO = 1,
    CALL = 2,
    CASH = 3,
    EQUITY = 4,
    REQUIRED = 5,
}

enum FinTransBasis{
	TRADE = 1,
	OFFSET = 2, 
	POSITIION = 3,
	OTHER_LEDGER = 4,
}

enum ScheduleType{
    IMMEDIATELY = 1,
    MANUAL = 2,
    DAILY = 3,
    MONTHLY = 4,
    YEARLY = 5,
	MATURITY = 6,
}

enum LedgerTypeID{
	CASH_TRN = 1,
	COMMISSION = 2,
	MAINT_MGN = 3,
	REAL_PNL = 4,
	BEGIN_BAL = 5,
	UNREAL_PNL = 6,
	END_BAL = 7,
	MGN_CALL = 8,
	EQUITY = 9,
	INTEREST = 10,
	COLLATERAL = 11,
	ADJUSTMENT = 12,
	SOD_EQUITY = 13,
	MTM_BEGIN_BAL = 14,
	MTM_REAL_PNL = 15,
	MTM_END_BAL = 16,
	MTM_UNREAL_PNL = 17,

	MONTHLY_REAL_PNL = 21,
	MONTHLY_INTEREST = 22,
	MONTHLY_CASH_TRN = 23,
	MONTHLY_BEGIN_BAL = 24,
	MONTHLY_COMMISSION = 25,
}

enum FinTransTypeID{
	CASH_TRN = 1,
	COMMISSION = 2,
	MAINT_MGN = 3,
	REAL_PNL = 4,
	BEGIN_BAL = 5,
	UNREAL_PNL = 6,
	END_BAL = 7,
	MGN_CALL = 8,
	EQUITY = 9,
	CARRYING_COST = 10,
	COLLATERAL_TRN = 11,
	ADJUSTMENT = 17,
	LONG_INTEREST = 24,
	SHORT_INTEREST = 25,
	DELIVERY_PAYMENT = 26,
	DELIVERY_FEE = 27,
    VALUE_ADDED_TAX = 28,
}

struct LedgerType{
	1: optional i32 id,
	2: required string code,
	3: optional string name,
    4: optional bool enabled,
    5: optional i32 ledgerGroup,

	6: optional bool sumToBalance,
	7: optional bool sumToEquity,
	8: optional bool sumToExcess,
	9: optional bool sumToMarginCall,
}

struct FinTransType{
	1: optional i32 id,
    2: required string code,
    3: optional string name,
    4: required bool enabled,
    5: ScheduleType scheduleType,

    6: optional i32 basis,
    7: optional string formula,
    8: required i32 ledgerTypeID,
    9: optional i32 srcLedgerTypeID,

    10: optional string memo,
    11: optional double minValue,
    12: optional double maxValue,
    13: optional bool byPercentage,

    14: optional string params,
    15: optional string paramDescs,
}

struct FinTrans{
	1: optional i32 id,
	2: required i32 tradeDate,
	3: required i32 ledgerTypeID,
	4: required i32 finTransTypeID,

	5: required double amount,
	6: required string account,
	7: required string currency,
	8: optional string accountPath,

	10: optional string transID,

	11: optional string exchange,
	12: optional string product,
	13: optional string instrument,
	
	14: optional string reference1,
	15: optional string reference2,
	16: optional string reference3,
	17: optional string reference4,
	18: optional i32 accountID,
	19: optional i32 currencyID,

	20: optional string memo,
	21: optional i64 modifiedTime,
	22: optional string modifiedBy,
}

struct DailyLedger{
	1: required i32 tradeDate,
	2: required string currency,
	3: required i32 accountID,
	4: required string account,
	5: optional string accountPath,

	10: optional double beginBalance,
	11: optional double cashTransfer,
	12: optional double collateral,
	13: optional double commission,
	14: optional double interest,
	15: optional double optionPremium,
	16: optional double realizedProfit,
	17: optional double forwardValue,
	18: optional double endingBalance,
	19: optional double unrealizedProfit,
	20: optional double equity,
	21: optional double netOptionValue,
	22: optional double initialMargin,
	23: optional double maintenanceMargin,
	24: optional double excess,
	25: optional double equityVariation,
	26: optional double swap, 
	27: optional double prevEquity,

	30: optional i64 modifiedTime,
}

enum RoleType{
	TRADER = 1,
	OFFICER = 2,
	AUDITOR = 3,
	SYS_USER = 4,
	ADMINISTRATOR = 5,
	BROKER_ADMINISTRATOR  = 6,
}

struct Permission{
	1: required string role,
	2: required string feature,

	3: required bool enableRead,
	4: required bool enableCreate,
	5: required bool enableUpdate,
	6: required bool enableDelete,
}

struct UserRole{
	1: required string code,
	2: optional string name,
	3: optional RoleType roleType,
	4: optional string parent,
	5: list<string> visibleTo,

	7: optional bool allowAll,
	8: optional bool allowOwn,
	9: optional bool allowChild,
}

struct User{
	1: optional i32 id,
	2: required string name,
	3: optional string password,

	4: required string role,
	5: optional string parent,
	6: optional UserStatus userStatus,

	7: optional i32 passwordExpireDate,
	8: optional i32 logonFailureCounter,
	9: optional bool requiredChangePassword,

	10: list<UserPrivilege> userPrivileges,
    11: optional i32 orderTypeMask,
    12: optional i32 platformMask,
	13: optional string allowIP,
	
	14: optional i64 passwordUpdateTime,
}

struct UserSession{
	1: optional i32 id,
	2: optional string user,
	3: optional UserStatus userStatus,
	4: optional string server,
	5: optional string platform,
	6: optional string logonIP,
	7: optional i32 port,
	8: optional i64 beginTime,
	9: optional i64 endTime,
	10: optional string sessionID,
}

struct AccountGroup{
	1: optional i32 id,
	2: optional string code,
	3: optional string name,
	4: optional string parent,
	5: optional double flatLevel,
	6: optional double warningLevel,
	7: optional double marginCallRatio,
    8: optional double marginScaleFactor,
}

struct Account{
    1: optional i32 id,
	2: required string code,
	3: optional string name,
	4: optional string path,
	5: optional string group,
    6: optional string sales,
	7: optional string parent,
	8: required string currency,
    9: required AccountType accountType,

  	10: optional bool enabled,
	11: optional bool autoFlat,
	12: optional bool checkLimit,
	13: optional bool checkCredit,
	14: optional bool enableOpenPosition,
    15: optional OffsetMethod offsetMethod,

    20: optional double lostLimit,
	21: optional double collateral,
	22: optional double beginBalance,
	23: optional double cashMovement,
	24: optional double totalLiquidity,

	25: optional double sodnlv,
	26: optional double equity,
	27: optional double profit,
	28: optional double fees,
	29: optional double optionValue,
	30: optional double realizedProfit,
    31: optional double unrealizedProfit,

	32: optional double maxRisk,
	33: optional double margin,
	34: optional double excess,
	35: optional double freeMargin,
	36: optional double marginRatio,

	40: optional string notes,
	41: optional i64 updateTime,
	42: optional string updateBy,
	43: optional bool isAggregated,
	44: optional i64 statementTime,
}

enum KycStatus{
	COMPLETED = 0,
	NOT_STARTED,
	KYC_CHECKING,
	KYC_CHECK_FAILURE,
	KYC_CHECK_COMPLETED,
	AML_CHECKING,
	AML_CHECK_FAILURE,
	AML_CHECK_COMPLETED,
	KYC2_CHECKING,
	KYC2_CHECK_FAILURE
}

struct AccountProfile{
	1: required string account,
	2: optional string name,
	3: optional string idNumber,
	4: optional Gender gender,
	5: optional i32 dateOfBirth,
	6: optional string occupation,
	7: optional string taxID,
	8: optional KycStatus kycStatus,

	10: optional string country,
	11: optional string city,
	12: optional string address1,
	13: optional string address2,
	14: optional string postcode,

	15: optional string email,
	16: optional string phone,
	17: optional string cellphone,

	21: optional string description,

	30: required list<Document> documents,
	31: required map<string, string> altIDs, //alternative ids, id name, id 
}

struct AccountAttribute{
	1: required string account,
	2: required string group,
	3: required string key,
	4: required string value,
	5: optional DataType type,
}

struct ProductSubscription{
	1: required string user,
	2: required string productGroup,
	3: required i32 userID,
	4: required i32 productGroupID,
}

struct AccountSubscription{
	1: required string user,
	2: required string account,
	3: required i32 userID,
	4: required i32 accountID,
}

struct AccountGroupSubscription{
	1: required string user,
	2: required string accountGroup,
	3: required i32 userID,
	4: required i32 accountGroupID,
}

struct AccountLimit{
    1: optional i32 id,
    2: optional string account,
    3: required string product,
    4: optional string instrument,
	
    5: optional i32 accountID,
    6: required i32 productID,
    7: optional i32 instrumentID,

	11: optional i32 longLimit,
	12: optional i32 shortLimit,
	13: optional i32 maxOrderSize,
	14: optional i32 intradayLimit,
	15: optional i32 exposureLimit,
}

struct LimitUtilization{
	1: required string account,
	2: required string product,
	3: required string instrument,
	4: required string accountGroup,

	11: optional i32 longLimit,
	12: optional i32 shortLimit,
	14: optional i32 intradayLimit,
	15: optional i32 exposureLimit,

	21: optional i32 sodNetQty,
	22: optional i32 sodLongQty,
	23: optional i32 sodShortQty,
	
	24: optional i32 dayNetQty,
	25: optional i32 dayLongQty,
	26: optional i32 dayShortQty,

	27: optional i32 netQty,
	28: optional i32 totalLongQty,
	29: optional i32 totalShortQty,

	30: optional i32 maxLimitExcess,
	31: optional i32 longLimitExcess,
	32: optional i32 shortLimitExcess,
	33: optional i32 intradayLimitExcess,
	34: optional i32 exposureLimitExcess,

	40: optional string currency,
	41: optional double profit,
	42: optional double profitInBaseCCY,

	51: optional i32 updateTime,
	52: optional string updateBy,
}

struct RatingClass{
    1: optional i32 id,
    2: required string code,
    3: required string name,
    5: required string parent,
    6: optional bool allowOverride,
}

struct Rating{
    1:  optional i32 id,
    2:  required i32 finTransType,
    //3:  optional i32 ratingClassID,

    4:  optional i32 accountID,
    5:  optional i32 accountGroupID,

    6:  optional i32 exchangeID,
    7:  optional i32 productID,
    8:  optional i32 instrumentID,

    9:  optional string currency,

    10: optional i32 tier,
    11: optional bool byPercentage,

    12: optional double minValue,
    13: optional double maxValue,

    14: optional double rate1,
    15: optional double rate2,
    16: optional double rate3,
    17: optional double rate4,
    18: optional double rate5,

    20: optional i32 validFrom,
    21: optional i32 validUntil,

    22: optional string account,
    23: optional string accountGroup,

    24: optional string exchange,
    25: optional string product,
    26: optional string instrument,
}

struct UserSetting{
	1: required string user,
	2: required string key,
	3: required string value,
	4: optional string description,
}

struct UserActivity{
	1: optional i64 id,
	2: optional i64 time,
	3: optional string user,
	4: optional string address,
	5: optional string feature,
	6: optional string accessType,
	7: optional string accessDetails,
	8: optional string applicationName,
}

enum TradeReportStatus{
	PRODUCT_NOT_FOUND = 1,
	ACCOUNT_NOT_FOUND = 2,
	UNCONFIRM = 5,
	CONFIRMED = 6,
}

struct TradeReport{
	1:  optional i32 id,
	2:  optional string platform,
	3:  required string reportID,

	4:  optional string account,
	5:  optional string exchange,
	6:  optional string product,
	7:  optional string instrument,
	8:  optional SecurityType securityType,

	9:  required Side side,
	10: required i32 qty,
	11: required double price,
	12: optional string execID,
	13: optional i64 transactTime,

	14: optional string mic,
	15: optional string symbol,
	16: optional string maturity,
	17: optional double strikePrice,
	18: optional TradeReportStatus tradeReportStatus,

	19: optional i32 tradeDate,
	20: optional i32 clearDate,

	21: optional string user,
	22: optional string orderID,
	23: optional string strategy,
	24: optional string currency,
	25: optional string clearFirm,
	26: optional string clearAccount,

	27: optional string notes,
	28: optional i64 modifiedTime,
	29: optional string modifiedBy,
	30: optional string reconcileID,

	31: optional bool threeWayTrade,

	// Product Details
	32: optional double tickSize,
	33: optional double tickValue,
	34: optional byte decimalPlace,
	35: optional byte qtyExponent,

	50: optional string exchAcro,
	51: optional string combCode,
	52: optional string prodCode,
	53: optional string prodType,
	54: optional i32 period,
	55: optional i32 optPeriod,
	56: optional i32 optStrikePrice,
	57: optional double contractValueFactor,
	58: optional byte openCloseFlag,
	59: optional byte speculationType,
}

struct Position{
	1: required string account,
	2: required string instrument,		
	
	3: optional i32 accountID,
	4: optional i32 instrumentID,
	5: optional i32 strategyID,
	
	6: optional double longPrice,
	7: optional i32 longQty,
	8: optional double shortPrice,
	9: optional i32 shortQty,
	
	10: optional byte decimalPlace,	

	11: optional i32 sodQty,
	12: optional double sodPrice,
	13: optional double sodProfit,

	14: optional i32 tradeQty,
	15: optional double tradePrice,
	16: optional double tradeProfit,

	17: optional i32 netQty,
	18: optional double avgPrice,

	19: optional double profit,
	20: optional double realizedProfit,
	21: optional double unrealizedProfit,	
	22: optional double profitInBaseCCY,

	23: optional double maxRisk,
	24: optional double marketPrice,
	25: optional double strikePrice,
	
	29: optional byte qtyExponent,
	
	30: optional string exchange,
	31: optional string product,
    32: optional string currency,
	33: optional string accountGroup,
	34: optional AccountType accountType,			

	35: optional double tickSize,
	36: optional double sodAmount,
	37: optional double tradeAmount,
	38: optional double netAmount,
	39: optional i32 contractSize,

	40: optional string notes,
	41: optional i32 updateTime,
	42: optional string updateBy,
	43: optional bool isAggregated,
	44: optional i32 tradeDate,	
	
	45: optional i64 exitOrderID,
	46: optional double maxLoss,
	47: optional double maxProfit,	

	50: optional string exchAcro,
	51: optional string combCode,
	52: optional string prodCode,
	53: optional string prodType,
	54: optional i32 period,
	55: optional i32 optPeriod,
	56: optional i32 optStrikePrice,
	57: optional double contractValueFactor,
	58: optional SecurityType securityType,
	59: optional string spanContract,
}

struct CurrencyRisk{
	1: required string code,
	2: required string account,
	3: optional double fxRate,
	4: optional double sodnlv,
	5: optional double profit,
	6: optional double equity,
	7: optional double collateral,
	8: optional double cashMovement,
	9: optional double profitInBaseCCY,

	10: optional double margin,
	11: optional double maxRisk,

	40: optional string notes,
	41: optional i64 updateTime,
	42: optional string updateBy,
}

struct ProductRisk{
	1: required string code,
	2: required string account,
	3: optional string exchange,
	4: optional string currency,

	11: optional byte scenarioIndex,
	12: optional double netDelta,
	13: optional double scanRisk,
	14: optional double spotCharge,
	15: optional double futurePriceRisk,
	16: optional double intraCommSpreadCharge,
	17: optional double interCommSpreadCredit,

	18: optional double shortPutQty,
	19: optional double shortCallQty,
	20: optional double netOptionValue,
	21: optional double shortOptionMinCharge,

	22: optional double profit,
	23: optional double profitInBaseCCY,
	24: optional double maxRisk,

	40: optional string notes,
	41: optional i64 updateTime,
	42: optional string updateBy,
}

struct AccountRiskView{
	1: optional i32 id,
	2: required string name,
	3: optional string description,
	4: required string owner,
	5: optional bool isShared,
	6: list<string> columnList,
	7: optional SearchCriteria searchCriteria,
}

struct ExecutionProvider{
	1: optional i32 id,
	2: required string code,
	3: optional string name,

	4: optional i32 ownerID,
	5: optional bool shared,
}

struct RoutingRule{
	1: optional i32 id,
	2: required string code,
	3: optional string name,

	4: optional i32 ownerID,
	5: optional i32 providerID,

	7: optional i32 exchangeID,
	8: optional i32 accountGroupID,
}

struct Fill{
	1: optional i32 id,
	2: optional i32 tradeDate,
	3: optional i32 orderID,
	4: optional i32 accountID,
	5: optional i32 strategyID,
	6: optional i32 instrumentID,
	7: optional byte decimalPlace,
	
	8: optional Side side,
	9: required i32 qty,
	10: optional i32 leavesQty,
	11: required double price,

	12: optional double settlementPrice,
	13: optional double unrealizedProfit,
	
	14: optional i64 transactTime,
	15: optional string executionID,

	16: optional string currency,
	17: optional string account,
	18: optional string exchange,
	19: optional string product,
	20: optional string instrument,
	21: optional string accountPath,
	22: optional string user,

	23: optional i32 fillDate,
	30: optional bool intradayTrade,
	
	31:optional double fees,
	32:optional double profit,
	33:optional double commission,
	34:optional byte qtyExponent,
	35:optional i32 orderInfoMask,
}

struct FillOffset{
	1: required i32 id,
	3: required i32 tradeDate,
	4: optional i32 orderID,
	5: optional i32 accountID,
	6: optional i32 stragegyID,
	7: optional i32 instrumentID,
	8: optional byte decimalPlace,

	9: required i32 offsetQty,
	10: required double profit,

	16: required string currency,
	17: required string account,
	18: required string exchange,
	19: required string instrument,
	20: optional string accountPath,

	21: optional i32 openFillID,
	22: optional i32 openFillDate,
	23: required i32 openFillQty,
	24: required double openFillPrice,
	25: optional i64 openFillTime,
	26: optional i32 openFillLeavesQty,

	31: optional i32 offsetFillID,
	32: optional i32 offsetFillDate,
	33: required i32 offsetFillQty,
	34: required double offsetFillPrice,
	35: optional i64 offsetFillTime,
	36: optional i32 offsetFillLeavesQty,
}

enum MonitorType{
	ACCOUNT,
	POSITION,
	INSTRUMENT,
}

enum ConditionType{
	//Account  Risk
	SODNLV = 0,
	EQUITY,
	PROFIT,
	MAXRISK,
	MARGIN,
	EXCESS,
	EQUITY_MARGIN,
	PROFIT_SODNLV,

	//Position Risk
	LONG_QTY = 30,
	SHORT_QTY,
	NET_POSITION,
	GROSS_POSITION, 

	//Market Risk
	TOTAL_TRADED = 50,
	PRICE_CHANGE,
	OPEN_INTEREST,
}

struct Condition{
	1: required ConditionType conditionType,
	2: required Comparator comparator,
	3: required double comparand,
}

struct RiskMonitor{
	1: optional i32 id,
	2: required string name,
	3: optional bool enabled,
	4: required MonitorType type,
	5: optional string description,
	6: required Condition condition,
	7: optional string owner,

	10: optional string group,
	11: optional string account,
	12: optional string exchange,
	13: optional string product,
	14: optional string instrument,
	15: optional AccountType accountType,
	16: optional SecurityType securityType,

	30: optional i32 priority,
	31: optional bool triggered,
	32: optional i64 triggeredTime,
	33: optional string lastAlertUUID,
	34: optional string emailRecipient,

	40: optional string notes,
	41: optional i64 updateTime,
	42: optional string updateBy,
}

enum RiskAlertStatus{
	NEW,
	VIEWED,
	ARCHIEVED,
}

struct RiskAlert{
	1: optional i32 id,
	2: required string uuid,
	3: optional i32 tradeDate,
	4: optional MonitorType type,
	5: optional string monitorName,
	6: optional Condition condition,
	7: optional string owner,
	8: optional double refValue,

	10: optional string group,
	11: optional string account,
	12: optional string exchange,
	13: optional string product,
	14: optional string instrument,
	15: optional AccountType accountType,
	16: optional SecurityType securityType,

	30: optional i32 priority,
	32: optional i64 triggeredTime,
	34: optional string emailRecipient,
	35: optional string alertSubject,
	36: optional string alertData,	

	40: optional string notes,
	41: optional i64 updateTime,
	42: optional string updateBy,
	43: optional string alertDetails,
	44: optional RiskAlertStatus status,
}

struct Collateral {
    1: optional i32 id,
    2: required string account,
    3: optional string collateralType,
    4: required double value,
    5: optional i32 valuationDate,
    6: optional double discount,
    7: optional string description,
}

struct News {
	1: optional i32 id,
    2: optional i64 time,
    3: required string subject,
    4: optional string category,
    5: optional string content,
}

struct StrategyLeg {
	1: required i32  legId,
	2: required Side legSide,
	3: required i32  legRatioQty, //qty multiplier
	4: optional i32  instrumentID,

	5: optional string exchange,
	6: optional string product,
	7: optional string instrument,
	8: optional double priceFactor,
}

struct Strategy {
	1: optional i32 id,
	2: required i32 userID,
	3: required string user, //owner
	4: required string name,
	5: list<StrategyLeg> legs,		
	6: optional string priceFormula,

	7: optional string currency,
	8: optional string exchange,
	9: optional string product,
}

struct BlockTrade{
	1: string uuid,
	2: string user,		
	3: string instrument,
	4: string buyerAccount,
	5: string sellerAccount,
	6: i64 transactTime,
	7: i32 instrumentID,
	8: i32 qty,
	9: double price, 
	10: string text,
}

struct Session{
	1: string uuid,
	2: string user,
	3: i64 lastActiveTime,
}

exception Exception{
	1: i32 errorCode,
	2: string errorMessage,
}

