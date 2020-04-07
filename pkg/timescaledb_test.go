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

func TestTimeScaledDB_Add(t *testing.T) {
	var db, err = pkg.NewTimeScaleDBFromURL(dbURL, tableName)
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
		require.Error(t, db.Add(context.Background(), rate))
	}

	t.Logf("Should only have one rate record in db")
	{
		var count, countErr = getTableCount(db.DB(), tableName)
		require.NoError(t, countErr)
		require.Equal(t, count, 1)
	}
}

func TestTimeScaledDB_AddBatch_WithDups(t *testing.T) {
	var db, err = pkg.NewTimeScaleDBFromURL(dbURL, tableName)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var rate btclists.Rate
	rate.Fiat = FIAT
	rate.Coin = COIN
	rate.Date = time.Now()
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
		require.Equal(t, count, 1)
	}
}

func TestTimeScaledDB_Latest(t *testing.T) {
	var db, err = pkg.NewTimeScaleDBFromURL(dbURL, tableName)
	require.NoError(t, err)
	require.NoError(t, prepareTestDatabase(db.DB()))

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var ts, terr = time.Parse(time.RFC3339, "2020-04-07T20:44:19+08:00")
	require.NoError(t, terr)
	require.False(t, ts.IsZero())

	var latest, lastErr = db.Latest(context.Background(), COIN, FIAT)
	require.NoError(t, lastErr)
	require.False(t, latest.Date.IsZero())
	require.True(t, latest.Date.Equal(ts))
}

func TestTimeScaledDB_Oldest(t *testing.T) {
	var db, err = pkg.NewTimeScaleDBFromURL(dbURL, tableName)
	require.NoError(t, err)
	require.NoError(t, prepareTestDatabase(db.DB()))

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var ts, terr = time.Parse(time.RFC3339, "2020-04-07T17:59:19+08:00")
	require.NoError(t, terr)
	require.False(t, ts.IsZero())

	var latest, lastErr = db.Oldest(context.Background(), COIN, FIAT)
	require.NoError(t, lastErr)
	require.False(t, latest.Date.IsZero())
	require.True(t, latest.Date.Equal(ts))
}

func TestTimeScaledDB_At(t *testing.T) {
	var db, err = pkg.NewTimeScaleDBFromURL(dbURL, tableName)
	require.NoError(t, err)
	require.NoError(t, prepareTestDatabase(db.DB()))

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var ts, terr = time.Parse(time.RFC3339, "2020-04-07T18:02:19+08:00")
	require.NoError(t, terr)
	require.False(t, ts.IsZero())

	var latestAt, lastErr = db.At(context.Background(), COIN, FIAT, ts)
	require.NoError(t, lastErr)
	require.Equal(t, 4, latestAt.Id)
	require.Equal(t, "523.121232", latestAt.Rate.String())
	require.Equal(t, ts, latestAt.Date)
}

func TestTimeScaledDB_Range(t *testing.T) {
	var db, err = pkg.NewTimeScaleDBFromURL(dbURL, tableName)
	require.NoError(t, err)
	require.NoError(t, prepareTestDatabase(db.DB()))

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var from, ferr = time.Parse(time.RFC3339, "2020-04-07T18:02:19+08:00")
	require.NoError(t, ferr)

	var to, terr = time.Parse(time.RFC3339, "2020-04-07T18:04:19+08:00")
	require.NoError(t, terr)

	var records, lastErr = db.Range(context.Background(), COIN, FIAT, from, to)
	require.NoError(t, lastErr)
	require.NotNil(t, records)
	require.Len(t, records, 3)

	var fixtures, fixtureErr = getFixtures()
	require.NoError(t, fixtureErr)
	require.Equal(t, []btclists.Rate{fixtures[5], fixtures[4], fixtures[3]}, records)
}

func TestTimeScaledDB_Avg(t *testing.T) {
	var db, err = pkg.NewTimeScaleDBFromURL(dbURL, tableName)
	require.NoError(t, err)
	require.NoError(t, prepareTestDatabase(db.DB()))

	defer func() {
		require.NoError(t, tearDownTable(db.DB(), tableName))
	}()

	var from, ferr = time.Parse(time.RFC3339, "2020-04-07T18:02:19+08:00")
	require.NoError(t, ferr)

	var to, terr = time.Parse(time.RFC3339, "2020-04-07T18:04:19+08:00")
	require.NoError(t, terr)

	var fixtures, fixtureErr = getFixtures()
	require.NoError(t, fixtureErr)

	var expectedAvg = decimal.NewFromFloat(0)
	expectedAvg = expectedAvg.Add(fixtures[5].Rate)
	expectedAvg = expectedAvg.Add(fixtures[4].Rate)
	expectedAvg = expectedAvg.Add(fixtures[3].Rate)
	expectedAvg = expectedAvg.Div(decimal.NewFromFloat(3))

	var avg, lastErr = db.AverageForRange(context.Background(), COIN, FIAT, from, to)
	require.NoError(t, lastErr)
	require.NotEmpty(t, expectedAvg, avg)
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
