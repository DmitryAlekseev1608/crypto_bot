package configs

const (
	USDT     string = "USDT"
	BYBIT    Market = "BYBIT"
	KUKOIN   Market = "KUKOIN"
	HTX      Market = "HTX"
	MEXC     Market = "MEXC"
	BINGX    Market = "BINGX"
	ASCENDEX Market = "ASCENDEX"
	BITGET   Market = "BITGET"
	XT       Market = "XT"
	BITMART  Market = "BITMART"
)

type Market string

func NewMarkets() []Market {
	return []Market{
		BYBIT,
		KUKOIN,
		HTX,
		MEXC,
		BINGX,
		ASCENDEX,
		BITGET,
		XT,
		BITMART,
	}
}
