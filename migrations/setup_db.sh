#!/bin/bash
set -e

echo "Starting migration setup script"

PGPASSWORD="$POSTGRES_PASSWORD" psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-SQL
    -- Create database
    CREATE DATABASE btc_listings owner $POSTGRES_USER;

    \connect btc_listings;

    -- NOT ANYMORE: Register timescaledb extension
    -- See https://docs.timescale.com/latest/getting-started/creating-hypertables
    -- CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;


    -- Create ratings table base to use for hypertable
    CREATE TABLE IF NOT EXISTS ratings (
        ID SERIAL NOT NULL,
        rate NUMERIC NOT NULL,
        coin VARCHAR(7) NOT NULL,
        fiat VARCHAR(7) NOT NULL,
        date TIMESTAMPTZ UNIQUE NOT NULL
    );

    -- Alter table to add primary keys, hyper table requires
    -- us to use time with it as well.
    ALTER TABLE ratings ADD PRIMARY KEY (id);
    ALTER TABLE ratings ADD CONSTRAINT fait_coin_date_unique unique (fiat, coin, date);
SQL

PGPASSWORD="$POSTGRES_PASSWORD" psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-SQL
    -- Create database
    CREATE DATABASE btc_listings_test owner $POSTGRES_USER;

    \connect btc_listings_test;

    -- NOT ANYMORE: Register timescaledb extension
    -- See https://docs.timescale.com/latest/getting-started/creating-hypertables
    -- CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

    -- Create ratings table base to use for hypertable
    CREATE TABLE IF NOT EXISTS ratings (
        ID SERIAL NOT NULL,
        rate NUMERIC NOT NULL,
        coin VARCHAR(7) NOT NULL,
        fiat VARCHAR(7) NOT NULL,
        date TIMESTAMPTZ UNIQUE NOT NULL
    );

    -- Alter table to add primary keys, hyper table requires
    -- us to use time with it as well.
    ALTER TABLE ratings ADD PRIMARY KEY (id);
    ALTER TABLE ratings ADD CONSTRAINT fait_coin_date_unique unique (fiat, coin, date);
SQL

echo "Finished running db migration script"
