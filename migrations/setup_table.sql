-- See https://docs.timescale.com/latest/getting-started/creating-hypertables

-- Create ratings table base to use for hypertable
CREATE TABLE IF NOT EXISTS ratings (
    ID SERIAL NOT NULL,
    time TIMESTAMPTZ NOT NULL,
    rate NUMERIC NOT NULL,
    coin VARCHAR(7) NOT NULL,
    fiat VARCHAR(7) NOT NULL
);


-- Alter table to add primary keys, hyper table requires
-- us to use time with it as well.
ALTER TABLE ratings ADD PRIMARY KEY (id, time);

-- Create index for coin and fiat as we will be using
-- these two a lot as well.
CREATE INDEX ON ratings (fiat, coin, time DESC);

-- Creates an hyper-table on top of ratings table
-- Using value in time column
SELECT create_hypertable('ratings', 'time');


