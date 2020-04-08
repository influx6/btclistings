package pkg_test

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/influx6/btclists"
	"github.com/shopspring/decimal"

	"github.com/influx6/btclists/pkg"

	"github.com/stretchr/testify/require"

	"github.com/go-testfixtures/testfixtures/v3"
)

var (
	tableName = "ratings"
	dbURL     = os.Getenv("DATABASE_URL")
)

func TestRatingsDB_Add(t *testing.T) {
	var db, err = pkg.NewPostgresDBFromURL(dbURL, tableName)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var rate btclists.Rate
	rate.Fiat = FIAT
	rate.Coin = COIN
	rate.Date = time.Now()
	rate.Rate = decimal.NewFromFloat(432.12)

	t.Logf("Should succesfully add rate record")
	{
		require.NoError(t, db.Add(context.Background(), rate))
	}

	t.Logf("Should fail to add duplicate rate record")
	{
		require.NoError(t, db.Add(context.Background(), rate))

		var count, countErr = getTableCount(db.DB(), tableName)
		require.NoError(t, countErr)
		require.Equal(t, 1, count)
	}
}

func TestRatingsDB_AddBatch_WithDups(t *testing.T) {
	var db, err = pkg.NewPostgresDBFromURL(dbURL, tableName)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var rate btclists.Rate
	rate.Fiat = FIAT
	rate.Coin = COIN
	rate.Date = time.Now().UTC()
	rate.Rate = decimal.NewFromFloat(432.12)

	var rates = []btclists.Rate{
		rate, rate, rate,
		rate, rate, rate,
		rate, rate, rate,
		rate, rate, rate,
	}

	t.Logf("Should succesfully add batch rate records")
	{
		require.NoError(t, db.AddBatch(context.Background(), rates))
	}

	t.Logf("Should only have one rate record in db")
	{
		var count, countErr = getTableCount(db.DB(), tableName)
		require.NoError(t, countErr)
		require.Equal(t, 1, count)
	}
}

func TestRatingsDB_Latest(t *testing.T) {
	var db, err = pkg.NewPostgresDBFromURL(dbURL, tableName)
	require.NoError(t, err)
	require.NoError(t, prepareTestDatabase(db.DB()))

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var fixtures, fixtureErr = getFixtures()
	require.NoError(t, fixtureErr)

	var target = fixtures[len(fixtures)-1]

	var latest, lastErr = db.Latest(context.Background(), COIN, FIAT)

	require.NoError(t, lastErr)
	require.False(t, latest.Date.IsZero())
	require.True(t, latest.Date.Equal(target.Date))
}

func TestRatingsDB_Oldest(t *testing.T) {
	var db, err = pkg.NewPostgresDBFromURL(dbURL, tableName)
	require.NoError(t, err)
	require.NoError(t, prepareTestDatabase(db.DB()))

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var fixtures, fixtureErr = getFixtures()
	require.NoError(t, fixtureErr)

	var target = fixtures[0]

	var oldest, lastErr = db.Oldest(context.Background(), COIN, FIAT)
	require.NoError(t, lastErr)
	require.False(t, oldest.Date.IsZero())
	require.True(t, oldest.Date.Equal(target.Date))
}

func TestRatingsDB_At(t *testing.T) {
	var db, err = pkg.NewPostgresDBFromURL(dbURL, tableName)
	require.NoError(t, err)
	require.NoError(t, prepareTestDatabase(db.DB()))

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var fixtures, fixtureErr = getFixtures()
	require.NoError(t, fixtureErr)

	var first = fixtures[0]

	var latestAt, lastErr = db.At(context.Background(), COIN, FIAT, first.Date)
	require.NoError(t, lastErr)
	require.Equal(t, 1, latestAt.Id)
	require.Equal(t, "6413.121232", latestAt.Rate.String())
	require.Equal(t, first.Date, latestAt.Date)
}

func TestRatingsDB_Range(t *testing.T) {
	var db, err = pkg.NewPostgresDBFromURL(dbURL, tableName)
	require.NoError(t, err)
	require.NoError(t, prepareTestDatabase(db.DB()))

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var fixtures, fixtureErr = getFixtures()
	require.NoError(t, fixtureErr)

	var fromRate = fixtures[3]
	var toRate = fixtures[5]

	var records, lastErr = db.Range(context.Background(), COIN, FIAT, fromRate.Date, toRate.Date)
	require.NoError(t, lastErr)
	require.NotNil(t, records)
	require.Len(t, records, 3)

	require.Equal(t, []btclists.Rate{fixtures[5], fixtures[4], fixtures[3]}, records)
}

func TestRatingsDB_AverageForRange(t *testing.T) {
	var db, err = pkg.NewPostgresDBFromURL(dbURL, tableName)
	require.NoError(t, err)
	require.NoError(t, prepareTestDatabase(db.DB()))

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var fixtures, fixtureErr = getFixtures()
	require.NoError(t, fixtureErr)

	var fromRate = fixtures[3]
	var toRate = fixtures[5]

	var expectedAvg = decimal.NewFromFloat(0)
	expectedAvg = expectedAvg.Add(fixtures[5].Rate)
	expectedAvg = expectedAvg.Add(fixtures[4].Rate)
	expectedAvg = expectedAvg.Add(fixtures[3].Rate)
	expectedAvg = expectedAvg.Div(decimal.NewFromFloat(3))

	var avg, lastErr = db.AverageForRange(context.Background(), COIN, FIAT, fromRate.Date, toRate.Date)
	require.NoError(t, lastErr)
	require.NotEmpty(t, expectedAvg, avg)
}

func TestRatingsDB_CountForRange(t *testing.T) {
	var db, err = pkg.NewPostgresDBFromURL(dbURL, tableName)
	require.NoError(t, err)
	require.NoError(t, prepareTestDatabase(db.DB()))

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var fixtures, fixtureErr = getFixtures()
	require.NoError(t, fixtureErr)

	var fromRate = fixtures[3]
	var toRate = fixtures[5]

	var count, lastErr = db.AverageForRange(context.Background(), COIN, FIAT, fromRate.Date, toRate.Date)
	require.NoError(t, lastErr)
	require.NotEmpty(t, 3, count)
}

func prepareTestDatabase(db *sql.DB) error {
	var fixtures, err = testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory("../fixtures"),
	)
	if err != nil {
		return err
	}

	return fixtures.Load()
}

func tearDownTable(db *sql.DB, table string) error {
	var _, err = db.Exec(fmt.Sprintf("TRUNCATE table %s", table))
	return err
}

func getTableCount(db *sql.DB, table string) (int, error) {
	var row = db.QueryRow(fmt.Sprintf("select count(*) from %s", table))

	var count int
	var err = row.Scan(&count)
	return count, err
}

func getFixtures() ([]btclists.Rate, error) {
	var fixtureData, err = ioutil.ReadFile("../fixtures/ratings.yml")
	if err != nil {
		return nil, err
	}

	var ratings []btclists.Rate
	if err := yaml.Unmarshal(fixtureData, &ratings); err != nil {
		return nil, err
	}

	return ratings, nil
}
