package postgres

import (
	"context"
	"crypto_pro/internal/adapters"
	"crypto_pro/internal/configs"
	"crypto_pro/internal/domain/entity"
	"crypto_pro/pkg/logger"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var _ adapters.DbAdapter = (*PostresRepository)(nil)
var OpErr *net.OpError

type PostresRepository struct {
	ctx    context.Context
	cfg    viper.Viper
	log    logger.Logger
	client *gorm.DB
}

func New(ctx context.Context, cfg viper.Viper, log logger.Logger) *PostresRepository {
	db := PostresRepository{
		ctx: context.Background(),
		cfg: cfg,
		log: log,
	}
	err := db.init()
	if err != nil {
		log.Panic("failed to init db", log.ErrorC(err))
	}
	return &db
}

func (d *PostresRepository) init() error {
	return d.repeatConnection(
		d.ctx,
		func() error {
			conn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
				d.cfg.GetString("postgres.host"), d.cfg.GetInt("postgres.port"), os.Getenv("POSTGRES_USER"),
				os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB"))
			client, err := gorm.Open(postgres.Open(conn), &gorm.Config{
				Logger: gormlogger.Default.LogMode(gormlogger.Silent),
			})
			if err != nil {
				d.log.Error("error create connection to DB", d.log.ErrorC(err))
				return err
			}
			d.client = client
			return nil
		},
		OpErr,
	)
}

func (d *PostresRepository) repeatConnection(ctx context.Context, fn func() error, errType error) error {
	var err error
	expBackOff := backoff.NewExponentialBackOff()
	expBackOff.MaxElapsedTime = 30 * time.Minute
	operation := func() error {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		err := fn()
		if errors.As(err, &errType) {
			return err
		} else if err != nil {
			return backoff.Permanent(err)
		}
		return nil
	}
	err = backoff.Retry(operation, expBackOff)
	if err != nil {
		return errors.Wrap(err, "failed init connection")
	}
	return nil
}

func (d *PostresRepository) Close() {
	client, err := d.client.DB()
	if err != nil {
		d.log.Error("error getting sql.DB from GORM: %v", d.log.ErrorC(err))
		return
	}
	client.Close()
}

func (d *PostresRepository) TrancateRowChains() error {
	tx := d.client.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	trancateQuery := `
		TRUNCATE TABLE raw_chains
	`

	if err := tx.Exec(trancateQuery).Error; err != nil {
		return err
	}

	return tx.Commit().Error
}

func (d *PostresRepository) InsertRowChains(chains []entity.Chain) error {
	if len(chains) == 0 {
		return nil
	}

	tx := d.client.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	var values []string
	var insertArgs []interface{}

	for i, chain := range chains {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)",
			i*6+1, i*6+2, i*6+3, i*6+4, i*6+5, i*6+6))
		insertArgs = append(insertArgs, chain.Market, chain.Symbol, chain.Chain,
			chain.WithdrawMin, chain.WithdrawMax, chain.WithdrawFee)
	}

	insertQuery := fmt.Sprintf(`
        INSERT INTO raw_chains (exchange, symbol, chain, withdraw_min, withdraw_max, withdraw_fee)
        VALUES %s
    `, strings.Join(values, ","))

	if err := tx.Exec(insertQuery, insertArgs...).Error; err != nil {
		return err
	}

	return tx.Commit().Error
}

func (d *PostresRepository) UpsertDWHChains() error {
	tx := d.client.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	var count int64
	if err := tx.Raw("SELECT COUNT(*) FROM raw_chains").Scan(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return tx.Commit().Error
	}

	deleteQuery := `
		DELETE FROM dwh_chains
		WHERE (symbol, chain) NOT IN (
			SELECT symbol, chain
			FROM raw_chains
			GROUP BY symbol, chain
			HAVING COUNT(*) >= 2
		)
	`

	if err := tx.Exec(deleteQuery).Error; err != nil {
		return err
	}

	insertQuery := `
		INSERT INTO dwh_chains (exchange, symbol, chain, withdraw_min, withdraw_max, withdraw_fee)
		SELECT
			r.exchange,
			r.symbol,
			r.chain,
			r.withdraw_min,
			r.withdraw_max,
			r.withdraw_fee
		FROM raw_chains r
		WHERE (r.symbol, r.chain) IN (
			SELECT symbol, chain
			FROM raw_chains
			GROUP BY symbol, chain
			HAVING COUNT(*) >= 2
		)
		ON CONFLICT (exchange, symbol, chain) DO UPDATE
		SET
			withdraw_min = EXCLUDED.withdraw_min,
			withdraw_max = EXCLUDED.withdraw_max,
			withdraw_fee = EXCLUDED.withdraw_fee,
			updated_at = CURRENT_TIMESTAMP
	`

	if err := tx.Exec(insertQuery).Error; err != nil {
		return err
	}

	return tx.Commit().Error
}

func (d *PostresRepository) UpsertDWHOrders() error {
	tx := d.client.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	var count int64
	if err := tx.Raw("SELECT COUNT(*) FROM dwh_chains").Scan(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		if err := tx.Exec("DELETE FROM dwh_orders").Error; err != nil {
			return err
		}
		return tx.Commit().Error
	}

	deleteQuery := `
		WITH exchange_symbol AS (
			SELECT exchange, symbol
			FROM dwh_chains
			GROUP BY exchange, symbol
		)
		DELETE FROM dwh_orders
		WHERE (exchange, symbol) NOT IN (
			SELECT exchange, symbol
			FROM exchange_symbol
		)
	`

	if err := tx.Exec(deleteQuery).Error; err != nil {
		return err
	}

	insertQuery := `
		WITH exchange_symbol AS (
			SELECT exchange, symbol
			FROM dwh_chains
			GROUP BY exchange, symbol
		)
		INSERT INTO dwh_orders (exchange, symbol)
		SELECT
			e.exchange,
			e.symbol
		FROM exchange_symbol e
		ON CONFLICT (exchange, symbol) DO UPDATE
		SET
			updated_at = CURRENT_TIMESTAMP
	`

	if err := tx.Exec(insertQuery).Error; err != nil {
		return err
	}

	return tx.Commit().Error
}

func (d *PostresRepository) SelectSymbols(market configs.Market) []string {
	var symbols []string
	tx := d.client.Begin()
	if tx.Error != nil {
		d.log.Error("error begin transaction", d.log.ErrorC(tx.Error))
		return nil
	}
	defer tx.Rollback()

	if err := tx.Raw("SELECT symbol FROM dwh_orders WHERE exchange = ?", market).Scan(&symbols).Error; err != nil {
		d.log.Error("error select symbols", d.log.ErrorC(err))
		return nil
	}

	if err := tx.Commit().Error; err != nil {
		d.log.Error("error commit transaction", d.log.ErrorC(err))
		return nil
	}

	return symbols
}

func (d *PostresRepository) UpdateOrders(data entity.Orders) error {
	tx := d.client.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer tx.Rollback()

	upsertQuery := `
		UPDATE dwh_orders
		SET
			ask_price = ?,
			bid_price = ?,
			updated_at = NOW()
		WHERE exchange = ? AND symbol = ?
	`

	askJSON, err := json.Marshal(data.AskPrice)
	if err != nil {
		d.log.Error("error marshal ask price", d.log.ErrorC(err))
		askJSON = []byte{}
	}
	bidJSON, err := json.Marshal(data.BidPrice)
	if err != nil {
		d.log.Error("error marshal bid price", d.log.ErrorC(err))
		bidJSON = []byte{}
	}

	if err := tx.Exec(upsertQuery, askJSON, bidJSON, data.Market, data.Symbol).Error; err != nil {
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

func (d *PostresRepository) SelectDWHChains() ([]entity.Chain, error) {
	rows, err := d.client.Raw(`
		SELECT exchange, symbol, chain, withdraw_min, withdraw_max, withdraw_fee
		FROM dwh_chains
	`).Rows()
	if err != nil {
		d.log.Error("error select all from dwh_chains", d.log.ErrorC(err))
		return nil, err
	}
	defer rows.Close()

	var chains []entity.Chain
	for rows.Next() {
		var chain entity.Chain
		if err := rows.Scan(&chain.Market, &chain.Symbol, &chain.Chain, &chain.WithdrawMin, &chain.WithdrawMax,
			&chain.WithdrawFee); err != nil {

			d.log.Error("error scanning row from dwh_chains", d.log.ErrorC(err))
			continue
		}
		chains = append(chains, chain)
	}

	return chains, nil
}

func (d *PostresRepository) SelectDWHOrders() ([]entity.Orders, error) {
	rows, err := d.client.Raw(`
		SELECT exchange, symbol, ask_price, bid_price
		FROM dwh_orders
	`).Rows()
	if err != nil {
		d.log.Error("error select all from dwh_orders", d.log.ErrorC(err))
		return nil, err
	}
	defer rows.Close()

	var orders []entity.Orders
	for rows.Next() {
		var order entity.Orders
		var askPriceJSON, bidPriceJSON []byte

		if err := rows.Scan(&order.Market, &order.Symbol, &askPriceJSON, &bidPriceJSON); err != nil {
			d.log.Error("error scanning row from dwh_orders", d.log.ErrorC(err))
			continue
		}

		if len(askPriceJSON) > 0 {
			if err := json.Unmarshal(askPriceJSON, &order.AskPrice); err != nil {
				d.log.Error("error unmarshaling ask_price JSON", d.log.ErrorC(err))
				continue
			}
		}

		if len(bidPriceJSON) > 0 {
			if err := json.Unmarshal(bidPriceJSON, &order.BidPrice); err != nil {
				d.log.Error("error unmarshaling bid_price JSON", d.log.ErrorC(err))
				continue
			}
		}

		orders = append(orders, order)
	}

	return orders, nil
}
