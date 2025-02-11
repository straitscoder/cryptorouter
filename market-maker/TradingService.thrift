include "Types.thrift"

namespace cpp Compass.Trading
namespace java com.compass.trading

service TradingService{
    i32 getTradeDate() throws (1: Types.Exception ex),
    i64 getServerTime() throws (1: Types.Exception ex),
    i32 getServerTimeZone() throws (1: Types.Exception ex),
	
	void delBlob(1:string key) throws (1: Types.Exception ex),
	string getBlob(1:string key) throws (1: Types.Exception ex),
	void setBlob(1:string key, 2:string blob) throws (1: Types.Exception ex),

	Types.Session login(1:string user, 2:string password, 3: string platform, 4: string ipAddr 5: i32 port) throws (1: Types.Exception ex),
	i32 touchSession(1: string sessionID) throws (1: Types.Exception ex),
	i32 logout(1:string sessionID) throws (1: Types.Exception ex),

	i32 placeOrder(1:Types.Order order) throws (1: Types.Exception ex),
	i32 amendOrder(1:Types.Order order) throws (1: Types.Exception ex),
	i32 cancelOrder(1:Types.Order order) throws (1: Types.Exception ex),
	i32 registerBlockTrade(1: Types.BlockTrade trade) throws (1: Types.Exception ex),

	list<string> getExchanges() throws (1: Types.Exception ex),
	list<Types.Currency> getCurrencies() throws (1: Types.Exception ex),
	Types.Instrument getInstrument(1: i32 instrumentID) throws (1: Types.Exception ex),
	list<Types.Instrument> getMyWatchList(1:string user) throws (1: Types.Exception ex),
	list<Types.Instrument> getInstruments(1: Types.SearchCriteria criteria) throws (1: Types.Exception ex),

	list<string> getInstrumentGroups(1: string user) throws (1: Types.Exception ex),
	list<Types.Instrument> getInstrumentByGroup(1: string group) throws (1: Types.Exception ex),

	bool addToMyWatchList(1:string user, 2:i32 instrumentID) throws (1: Types.Exception ex),
	bool removeFromMyWatchList(1:string user, 2:i32 instrumentID) throws (1: Types.Exception ex),

	Types.Quote getQuote(1: i32 instrumentID) throws (1: Types.Exception ex),
	list<Types.Quote> getMyQuotes(1: string user) throws (1: Types.Exception ex),
	list<Types.Quote> getQuotes(1: list<Types.QuoteVersion> items) throws (1: Types.Exception ex),
	list<Types.Quote> getQuoteList(1: Types.SearchCriteria criteria) throws (1: Types.Exception ex),
	map<i32, Types.Quote> getQuoteMap(1: list<Types.QuoteVersion> items) throws (1: Types.Exception ex),

	Types.Bars getNBars(1:i32 instrumentID, 2:Types.BarInterval interval, 3:i32 noOfBars) throws (1: Types.Exception ex),
	Types.Bars getBarsSince(1:i32 instrumentID, 2:Types.BarInterval interval, 3:i64 since) throws (1: Types.Exception ex),

	list<Types.News> getNews(1: Types.SearchCriteria criteria) throws (1: Types.Exception ex),

	Types.Order getOrder(1:i32 orderID) throws (1: Types.Exception ex),
	list<Types.Fill> getFills(1: Types.SearchCriteria criteria) throws (1: Types.Exception ex),	
	list<Types.Order> getOrders(1: Types.SearchCriteria criteria) throws (1: Types.Exception ex),
	list<Types.Order> getOrderHistory(1: Types.SearchCriteria criteria) throws (1: Types.Exception ex),	

	Types.Account getAccount(1:i32 accountID) throws (1: Types.Exception ex),
	Types.Account getDefaultAccount(1:string user) throws (1: Types.Exception ex),
	Types.AccountProfile getAccountProfile(1:string account) throws (1: Types.Exception ex),
	void setAccountProfile(1:Types.AccountProfile accountProfile) throws (1: Types.Exception ex),
	list<Types.Account>  getAccounts(1:Types.SearchCriteria criteria) throws (1: Types.Exception ex),
	list<Types.Position> getPositions(1:Types.SearchCriteria criteria) throws (1: Types.Exception ex),
	list<Types.Position> getChildPositions(1:Types.SearchCriteria criteria) throws (1: Types.Exception ex),
	
	list<Types.FinTransType> getFinTransTypes() throws (1: Types.Exception ex),
	list<Types.FinTrans> getFinTrans(1: Types.SearchCriteria criteria) throws (1: Types.Exception ex),
	list<Types.DailyLedger> getDailyLedger(1: Types.SearchCriteria criteria) throws (1: Types.Exception ex),

    bool isPasswordChangeRequred(1: string user) throws (1: Types.Exception ex),
	bool changePassword(1:string user, 2:string oldPassword, 3:string newPassword) throws (1: Types.Exception ex),

	map<string, string> getSettings(1:string user) throws (1: Types.Exception ex),
	string getSetting(1: string user, 2: string key) throws (1: Types.Exception ex),
	bool delSetting(1: string user, 2: string key) throws (1: Types.Exception ex),	
	bool saveSetting(1: string user, 2: string key, 3: string value) throws (1: Types.Exception ex),
	
	bool isUserRegistered(1:string username) throws (1: Types.Exception ex),
	void sendMail(1:string email, 2:string subject 3:string content) throws (1: Types.Exception ex),
	void sendMailEx(1:string recipient, 2: string subject, 3: string mail, 4: list<string> attachments) throws (1: Types.Exception ex),
	
    Types.Account registerAccount(1: string username, 2:string password, 3:string broker 4: string referrer) throws (1: Types.Exception ex),

    void requestWithdraw(1:string account, 2:string ccy, 3: double amount) throws (1: Types.Exception ex),

    list<Types.AccountAttribute> getAccountAttributes(1:string account) throws (1: Types.Exception ex),
    void delAccountAttribute(1:string account, 2:string group, 3:string key) throws (1: Types.Exception ex),
    string getAccountAttribute(1:string account, 2:string group, 3:string key) throws (1: Types.Exception ex),
    void setAccountAttribute(1:string account, 2:string tag, 3:string group, 4:string value) throws (1: Types.Exception ex),
	
	void setUser(1:Types.User user) throws (1: Types.Exception ex),
	void setQuote(1:Types.Quote quote) throws (1: Types.Exception ex),
    void setRating(1: Types.Rating rate) throws (1: Types.Exception ex),
    void delRating(1: Types.Rating rate) throws (1: Types.Exception ex),	
	void setAccount(1: Types.Account account) throws (1: Types.Exception ex),	
    void setSettlementPrice(1: Types.SettlementPrice price) throws (1: Types.Exception ex),	
	void setUserSession(1: Types.UserSession userSession) throws (1: Types.Exception ex),
	void archiveUserSession(1: Types.UserSession userSession) throws (1: Types.Exception ex),
    void setAccountLimit(1: Types.AccountLimit limit) throws (1: Types.Exception ex),	
    void delAccountLimit(1: string account, 2: string product, 3: string instrument) throws (1: Types.Exception ex),	
}
