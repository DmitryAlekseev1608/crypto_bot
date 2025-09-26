package postgres

import (
	"context"
	"crypto_pro/internal/adapters"
	"crypto_pro/internal/domain/entity"
	"crypto_pro/pkg/logger"
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

	if err := db.createConnection(cfg.GetString("postgres.host_local")); err != nil {
		log.Info("failed to init db in localhost", log.ErrorC(err))
		if err := db.createConnectionRemote(); err != nil {
			log.Panic("failed to init db in remote", log.ErrorC(err))
		}
	}

	return &db
}

func (d *PostresRepository) createConnectionRemote() error {
	return d.repeatConnection(
		d.ctx,
		func() error {
			if err := d.createConnection(d.cfg.GetString("postgres.host_remote")); err != nil {
				d.log.Error("error create connection to DB", d.log.ErrorC(err))
				return err
			}
			return nil
		},
		OpErr,
	)
}

func (d *PostresRepository) createConnection(host string) error {
	conn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host,
		d.cfg.GetInt("postgres.port"), os.Getenv("POSTGRES_USER"),
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

func (d *PostresRepository) UpsertDWHTransactions(transactionsEntity []entity.Transaction) error {
	tx := d.client.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	var values []string
	var insertArgs []interface{}

	transactionsModel, err := fromEntityToModel(transactionsEntity)
	if err != nil {
		return err
	}

	timeNow := time.Now()

	for i, transaction := range transactionsModel {
		values = append(values, fmt.Sprintf(`($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d,
		$%d, $%d, $%d, $%d, $%d, $%d)`,
			i*16+1, i*16+2, i*16+3, i*16+4, i*16+5, i*16+6, i*16+7, i*16+8, i*16+9, i*16+10,
			i*16+11, i*16+12, i*16+13, i*16+14, i*16+15, i*16+16))

		insertArgs = append(insertArgs, transaction.ID, transaction.Symbol, transaction.Chain,
			transaction.MarketFrom, transaction.MarketTo, transaction.Spread,
			transaction.WithDrawFee, transaction.WithdrawMax, transaction.AmountCoin,
			transaction.AmountAskOrder, transaction.AskCost, transaction.AskOrder,
			transaction.AmountBidOrder, transaction.BidCost, transaction.BidOrder,
			timeNow)
	}

	insertQuery := fmt.Sprintf(`
		INSERT INTO raw_transactions (id, symbol, chain, market_from, market_to, spread, 
			with_draw_fee, withdraw_max, amount_coin, amount_ask_order, ask_cost, ask_order,
			amount_bid_order, bid_cost, bid_order, updated_at)
		VALUES %s
	`, strings.Join(values, ","))

	if err := tx.Exec(insertQuery, insertArgs...).Error; err != nil {
		return err
	}

	deleteQuery := `
		DELETE FROM dwh_transactions
		WHERE (id, symbol, chain, market_from, market_to) NOT IN (
			SELECT id, symbol, chain, market_from, market_to
			FROM raw_transactions
		)
	`

	if err := tx.Exec(deleteQuery).Error; err != nil {
		return err
	}

	insertQuery = `
		INSERT INTO dwh_transactions (id, symbol, chain, market_from, market_to, spread, 
			with_draw_fee, withdraw_max, amount_coin, amount_ask_order, ask_cost, ask_order,
			amount_bid_order, bid_cost, bid_order, updated_at)
		SELECT
			r.id,
			r.symbol,
			r.chain,
			r.market_from,
			r.market_to,
			r.spread,
			r.with_draw_fee,
			r.withdraw_max,
			r.amount_coin,
			r.amount_ask_order,
			r.ask_cost,
			r.ask_order,
			r.amount_bid_order,
			r.bid_cost,
			r.bid_order,
			r.updated_at
		FROM raw_transactions r
		ON CONFLICT (id, symbol, chain, market_from, market_to) DO UPDATE
		SET
			spread = EXCLUDED.spread,
			with_draw_fee = EXCLUDED.with_draw_fee,
			withdraw_max = EXCLUDED.withdraw_max,
			amount_coin = EXCLUDED.amount_coin,
			amount_ask_order = EXCLUDED.amount_ask_order,
			ask_cost = EXCLUDED.ask_cost,
			ask_order = EXCLUDED.ask_order,
			amount_bid_order = EXCLUDED.amount_bid_order,
			bid_cost = EXCLUDED.bid_cost,
			bid_order = EXCLUDED.bid_order,
			updated_at = CURRENT_TIMESTAMP
	`

	if err := tx.Exec(insertQuery).Error; err != nil {
		return err
	}

	deleteQuery = fmt.Sprintf(`
		DELETE FROM raw_transactions
		WHERE id = '%s'
	`, transactionsModel[0].ID)

	if err := tx.Exec(deleteQuery).Error; err != nil {
		return err
	}

	return tx.Commit().Error
}

func (d *PostresRepository) SelectTransactions(id string) []entity.Transaction {
	var transactions transactions

	tx := d.client.Begin()
	if tx.Error != nil {
		d.log.Error("error begin transaction", d.log.ErrorC(tx.Error))
		return nil
	}
	defer tx.Rollback()

	if err := tx.Raw(`
		SELECT
			id,
			symbol,
			chain,
			market_from,
			market_to,
			spread,
			with_draw_fee,
			withdraw_max,
			amount_coin,
			amount_ask_order,
			ask_cost,
			ask_order,
			amount_bid_order,
			bid_cost,
			bid_order,
			updated_at
		FROM dwh_transactions
		WHERE id = $1`, id).Scan(&transactions).Error; err != nil {
		d.log.Error("error select transactions", d.log.ErrorC(err))
		return nil
	}

	if err := tx.Commit().Error; err != nil {
		d.log.Error("error commit transaction", d.log.ErrorC(err))
		return nil
	}

	return transactions.toEntity()
}

func (d *PostresRepository) TrancateRawTransactions() {
	if err := d.client.Exec("TRUNCATE TABLE raw_transactions").Error; err != nil {
		d.log.Error("error truncate raw_transactions", d.log.ErrorC(err))
	}

}

func (d *PostresRepository) TrancateDwhTransactions() {
	if err := d.client.Exec("TRUNCATE TABLE dwh_transactions").Error; err != nil {
		d.log.Error("error truncate dwh_transactions", d.log.ErrorC(err))
	}
}

func (d *PostresRepository) DeleteSession(id string) {
	deleteQuery := fmt.Sprintf(`
		DELETE FROM dwh_transactions
		WHERE id = '%s'
	`, id)

	if err := d.client.Exec(deleteQuery).Error; err != nil {
		d.log.Error("error delete session", d.log.ErrorC(err))
	}
}

func (d *PostresRepository) SelectTransactionsBySymbol(id string, symbol, marketFrom,
	marketTo string) entity.Transaction {

	var transaction transaction

	if err := d.client.Raw(`
		SELECT * FROM dwh_transactions WHERE id=$1 AND symbol=$2 AND market_from=$3 AND
			market_to=$4`, id, symbol, marketFrom, marketTo).Scan(&transaction).Error; err != nil {
		d.log.Error("error select transactions", d.log.ErrorC(err))
	}
	transactions := transactions{transaction}
	if len(transactions.toEntity()) == 0 {
		return entity.Transaction{}
	}
	return transactions.toEntity()[0]
}

func (d *PostresRepository) SelectNewTransactions(id string) []entity.Transaction {
	var transactions transactions

	tx := d.client.Begin()
	if tx.Error != nil {
		d.log.Error("error begin transaction", d.log.ErrorC(tx.Error))
		return nil
	}
	defer tx.Rollback()

	if err := tx.Raw(`
		SELECT
			id,
			symbol,
			chain,
			market_from,
			market_to,
			spread,
			with_draw_fee,
			withdraw_max,
			amount_coin,
			amount_ask_order,
			ask_cost,
			ask_order,
			amount_bid_order,
			bid_cost,
			bid_order,
			updated_at
		FROM dwh_transactions
		WHERE id = $1 AND is_posted = false`, id).Scan(&transactions).Error; err != nil {
		d.log.Error("error select transactions", d.log.ErrorC(err))
		return nil
	}

	if err := tx.Commit().Error; err != nil {
		d.log.Error("error commit transaction", d.log.ErrorC(err))
		return nil
	}

	d.updateTransactionIsPosted(id)

	return transactions.toEntity()
}

func (d *PostresRepository) updateTransactionIsPosted(id string) error {
	if err := d.client.Exec("UPDATE dwh_transactions SET is_posted = true WHERE id = ?", id).Error; err != nil {
		d.log.Error("error update transactions", d.log.ErrorC(err))
		return err
	}
	return nil
}
