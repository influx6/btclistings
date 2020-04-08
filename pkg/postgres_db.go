package pkg

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/shopspring/decimal"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"

	"github.com/Masterminds/squirrel"

	"github.com/influx6/btclists"
	"github.com/jackc/pgtype"
)

const (
	acceptableRange = 5 * time.Minute
)

type PostgresDB struct {
	db    *sql.DB
	table string
	sdb   squirrel.StatementBuilderType
}

func NewPostgresDB(db *sql.DB, table string) (*PostgresDB, error) {
	var sqdb = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(db)
	var tdb PostgresDB
	tdb.db = db
	tdb.sdb = sqdb
	tdb.table = table
	return &tdb, nil
}

func NewPostgresDBFromURL(dbURL string, table string) (*PostgresDB, error) {
	var c, err = pgx.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}

	var db = stdlib.OpenDB(*c)
	return NewPostgresDB(db, table)
}

func (t *PostgresDB) DB() *sql.DB {
	return t.db
}

func (t *PostgresDB) Close() error {
	return t.db.Close()
}

func (t *PostgresDB) Add(ctx context.Context, rate btclists.Rate) error {
	var rating, err = rate.Rate.Value()
	if err != nil {
		log.Printf("[ERROR] | [DB] | Failed transform decimal rating to db.Value | %s\n", err)
		return err
	}

	var q = t.sdb.Insert(t.table).
		Columns("date", "rate", "coin", "fiat").
		Values(
			rate.Date,
			rating,
			rate.Coin,
			rate.Fiat,
		)
	if _, err := q.ExecContext(ctx); err != nil {
		log.Printf("[ERROR] | [DB] | Failed insert record into db.Value | %s\n", err)
		return err
	}
	return nil
}

func (t *PostgresDB) AddBatch(ctx context.Context, rates []btclists.Rate) error {
	if len(rates) == 0 {
		return nil
	}

	var q = t.sdb.Insert(t.table).
		Columns("date", "rate", "coin", "fiat")

	for _, rate := range rates {
		var ratings, err = rate.Rate.Value()
		if err != nil {
			return err
		}
		q = q.Values(
			rate.Date.Format(btclists.DateTimeFormat),
			ratings,
			rate.Coin,
			rate.Fiat,
		)
	}

	q = q.Suffix(`
		ON CONFLICT (date) DO NOTHING
	`)
	if _, err := q.ExecContext(ctx); err != nil {
		return err
	}
	return nil
}

func (t *PostgresDB) Latest(ctx context.Context, coin string, fiat string) (btclists.Rate, error) {
	var q = t.sdb.
		Select("id", "date", "rate", "coin", "fiat").
		From(t.table).
		Where(squirrel.Eq{
			"coin": coin,
			"fiat": fiat,
		}).
		OrderBy("date DESC").
		Limit(1)

	var row = q.QueryRowContext(ctx)

	var ts pgtype.Timestamptz
	var rate btclists.Rate
	if err := row.Scan(&rate.Id, &ts, &rate.Rate, &rate.Coin, &rate.Fiat); err != nil {
		log.Printf("[ERROR] | [DB] | Failed to marshal row | %s\n", err)
		return rate, err
	}

	rate.Date = ts.Time

	return rate, nil
}

func (t *PostgresDB) Oldest(ctx context.Context, coin string, fiat string) (btclists.Rate, error) {
	var q = t.sdb.
		Select("id", "date", "rate", "coin", "fiat").
		From(t.table).
		Where(squirrel.Eq{
			"coin": coin,
			"fiat": fiat,
		}).
		OrderBy("date ASC").
		Limit(1)

	var row = q.QueryRowContext(ctx)
	var ts pgtype.Timestamptz
	var rate btclists.Rate
	if err := row.Scan(&rate.Id, &ts, &rate.Rate, &rate.Coin, &rate.Fiat); err != nil {
		log.Printf("[ERROR] | [DB] | Failed to marshal row | %s\n", err)
		return rate, err
	}

	rate.Date = ts.Time

	return rate, nil
}

// At tries to retrieve ratings at giving timestamp but if such rating for exactly the giving
// time is not available, then the next rating within a 1 min range would be returned.
func (t *PostgresDB) At(ctx context.Context, coin string, fiat string, tm time.Time) (btclists.Rate, error) {
	var q = t.sdb.
		Select("t.id", "t.date", "t.rate", "t.coin", "t.fiat").
		From(fmt.Sprintf("%s t", t.table)).
		Where(squirrel.Eq{
			"t.coin": coin,
			"t.fiat": fiat,
		}).
		Where(
			"t.date BETWEEN $3 AND $4",
			tm,
			tm.Add(acceptableRange), // scale this over 1 minutes, so we should be able to get exact or closest.
		).
		OrderBy("t.date ASC").
		Limit(1)

	var row = q.QueryRowContext(ctx)

	var rate btclists.Rate

	var ts pgtype.Timestamptz
	if err := row.Scan(&rate.Id, &ts, &rate.Rate, &rate.Coin, &rate.Fiat); err != nil {
		log.Printf("[ERROR] | [DB] | Failed to marshal row | %s\n", err)
		return rate, err
	}

	rate.Date = ts.Time
	return rate, nil
}

func (t *PostgresDB) Range(ctx context.Context, coin string, fiat string, from time.Time, to time.Time) ([]btclists.Rate, error) {
	var q = t.sdb.
		Select("t.id", "t.date", "t.rate", "t.coin", "t.fiat").
		From(fmt.Sprintf("%s t", t.table)).
		Where(squirrel.Eq{
			"t.coin": coin,
			"t.fiat": fiat,
		}).
		Where(
			"t.date BETWEEN $3 AND $4",
			from,
			to,
		).
		OrderBy("date DESC")

	var rows, err = q.QueryContext(ctx)
	if err != nil {
		log.Printf("[ERROR] | [DB] | Failed query request | %s\n", err)
		return nil, err
	}

	var rates []btclists.Rate
	for rows.Next() {
		var rate btclists.Rate

		var ts pgtype.Timestamptz
		if err := rows.Scan(&rate.Id, &ts, &rate.Rate, &rate.Coin, &rate.Fiat); err != nil {
			log.Printf("[ERROR] | [DB] | Failed scan row into struct | %s\n", err)
			return nil, err
		}

		rate.Date = ts.Time
		rates = append(rates, rate)
	}

	return rates, nil
}

func (t *PostgresDB) AverageForRange(ctx context.Context, coin string, fiat string, from time.Time, to time.Time) (decimal.Decimal, error) {
	var q = t.sdb.
		Select("AVG(rate)").
		From(t.table).
		Where(squirrel.Eq{
			"coin": coin,
			"fiat": fiat,
		}).
		Where(
			"date between $3 and $4",
			from,
			to,
		)

	var avg decimal.Decimal

	var row = q.QueryRowContext(ctx)
	if err := row.Scan(&avg); err != nil {
		log.Printf("[ERROR] | [DB] | Failed to marshal row | %s\n", err)
		return avg, err
	}

	return avg, nil
}

func (t *PostgresDB) CountForRange(ctx context.Context, coin string, fiat string, from time.Time, to time.Time) (int, error) {
	var q = t.sdb.
		Select("Count(*)").
		From(t.table).
		Where(squirrel.Eq{
			"coin": coin,
			"fiat": fiat,
		}).
		Where(
			"date between $3 and $4",
			from,
			to,
		)

	var total int

	var row = q.QueryRowContext(ctx)
	if err := row.Scan(&total); err != nil {
		log.Printf("[ERROR] | [DB] | Failed to marshal row | %s\n", err)
		return total, err
	}

	return total, nil
}
