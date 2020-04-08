# BTCListings
BTCListings provides a simple example project for a single BTC to USD exchange rate service.

## Platform Requirements

- PostgreSQL (Docker Image)
- Docker and docker-compose
- Go (go version go1.14.1 darwin/amd64)

## Chosen DB

I initially considered [InfluxDB](https://www.influxdata.com/) as I considered the nature of the project
geared towards time series, which also lead me to [TimescaleDB](https://www.timescale.com/) as an interesting alternative
on top of PostgreSQL, but the more I considered the architecture, it was evident a simple PostgreSQL database
would do fine as the fine-grained time queries provided by either option (InfluxDB and TimeScaleDB) were not needed.

## Chosen API

I chose to use [Coin API](https://www.coinapi.io/) as the rating API integration for 
the project. So you will need to provide this as either an environment variable `COIN_API_TOKEN`
or as part of a `.env` file.

Coin API seems to have a hard requirements on request hit based on accounts, so might quickly hit limit
on a per-day basis if using the Free account as I did.

## Easiest Local Run

With docker and docker-compose installed and the CoinAPI token key received.
Simply create a `.env` file to host environment variables used by the application.

*This is required for both development, production and when executing tests, if you prefer feeding these one by one using `env VAR=VAL` then this will work as well.*

Add the following to an '.env' file:

```bash
# set database url for use by docker image
DATABASE_URL=postgres://postgres:starcraft@db:5432/btc_listings

# set coin api token 
COIN_API_TOKEN=26******-**********-*********-5D5D

# set host and port for go server
HOST=0.0.0.0

PORT=80
```

Then bootup db and server with:

```bash
make up
```

You should be ready to hit the API with requests once db has finished setup and server has connected successfully to it.

Beyond requiring docker-compose and docker (which most do have), with that simple command,
the necessary server and database would be booted up ready for testing. Do note that due to 
not setting up `a wait till db is ready before running server` type of script, docker-compose will
attempt to restart the server until it can successfully connect to the database, as the database
setups and executes migration scripts to prepare the db and table.

See sample run below:

```bash
12:22:16 alexewetumo@GINI-0023 btclistings ±|master ✗|→ make up
docker-compose -f docker-compose.local.yml up
Starting postgre ... done
Starting btc_listings ... done
Attaching to postgre, btc_listings
postgre | The files belonging to this database system will be owned by user "postgres".
postgre | This user must also own the server process.
postgre |
postgre | The database cluster will be initialized with locale "en_US.utf8".
postgre | The default database encoding has accordingly been set to "UTF8".
postgre | The default text search configuration will be set to "english".
postgre |
postgre | Data page checksums are disabled.
postgre |
postgre | fixing permissions on existing directory /var/lib/postgresql/data ... ok
btc_listings | 2020/04/08 16:22:19 [BTC Listings] | Failed to verify database connection: failed to connect to `host=db user=postgres database=btc_listings`: dial error (dial tcp 172.21.0.2:5432: connect: connection refused)
postgre | creating subdirectories ... ok
postgre | selecting dynamic shared memory implementation ... posix
postgre | selecting default max_connections ... 100
postgre | selecting default shared_buffers ... 128MB
postgre | selecting default time zone ... Etc/UTC
postgre | creating configuration files ... ok
btc_listings exited with code 1
btc_listings exited with code 1
btc_listings exited with code 1
btc_listings exited with code 1
postgre | running bootstrap script ... ok
btc_listings exited with code 1
btc_listings exited with code 1
postgre | performing post-bootstrap initialization ... ok
postgre | syncing data to disk ... ok
postgre |
postgre |
postgre | Success. You can now start the database server using:
postgre |
postgre |     pg_ctl -D /var/lib/postgresql/data -l logfile start
postgre |
postgre | initdb: warning: enabling "trust" authentication for local connections
postgre | You can change this by editing pg_hba.conf or using the option -A, or
postgre | --auth-local and --auth-host, the next time you run initdb.
postgre | waiting for server to start....2020-04-08 16:22:34.128 UTC [49] LOG:  starting PostgreSQL 12.2 (Debian 12.2-2.pgdg100+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 8.3.0-6) 8.3.0, 64-bit
postgre | 2020-04-08 16:22:34.131 UTC [49] LOG:  listening on Unix socket "/var/run/postgresql/.s.PGSQL.5432"
postgre | 2020-04-08 16:22:34.195 UTC [50] LOG:  database system was shut down at 2020-04-08 16:22:31 UTC
postgre | 2020-04-08 16:22:34.223 UTC [49] LOG:  database system is ready to accept connections
postgre |  done
postgre | server started
postgre |
postgre | /usr/local/bin/docker-entrypoint.sh: running /docker-entrypoint-initdb.d/setup_db.sh
postgre | Starting migration setup script
btc_listings exited with code 1
postgre | CREATE DATABASE
postgre | You are now connected to database "btc_listings" as user "postgres".
postgre | CREATE TABLE
postgre | ALTER TABLE
postgre | ALTER TABLE
postgre | CREATE DATABASE
postgre | You are now connected to database "btc_listings_test" as user "postgres".
postgre | CREATE TABLE
postgre | ALTER TABLE
postgre | ALTER TABLE
postgre | Finished running db migration script
postgre |
postgre | 2020-04-08 16:22:38.930 UTC [49] LOG:  received fast shutdown request
postgre | waiting for server to shut down...2020-04-08 16:22:38.932 UTC [49] LOG:  aborting any active transactions
postgre | 2020-04-08 16:22:38.934 UTC [49] LOG:  background worker "logical replication launcher" (PID 56) exited with exit code 1
postgre | .2020-04-08 16:22:38.945 UTC [51] LOG:  shutting down
postgre | 2020-04-08 16:22:39.057 UTC [49] LOG:  database system is shut down
postgre |  done
postgre | server stopped
postgre |
postgre | PostgreSQL init process complete; ready for start up.
postgre |
postgre | 2020-04-08 16:22:39.181 UTC [1] LOG:  starting PostgreSQL 12.2 (Debian 12.2-2.pgdg100+1) on x86_64-pc-linux-gnu, compiled by gcc (Debian 8.3.0-6) 8.3.0, 64-bit
postgre | 2020-04-08 16:22:39.181 UTC [1] LOG:  listening on IPv4 address "0.0.0.0", port 5432
postgre | 2020-04-08 16:22:39.182 UTC [1] LOG:  listening on IPv6 address "::", port 5432
postgre | 2020-04-08 16:22:39.186 UTC [1] LOG:  listening on Unix socket "/var/run/postgresql/.s.PGSQL.5432"
postgre | 2020-04-08 16:22:39.281 UTC [79] LOG:  database system was shut down at 2020-04-08 16:22:39 UTC
postgre | 2020-04-08 16:22:39.316 UTC [1] LOG:  database system is ready to accept connections
btc_listings | 2020/04/08 16:22:48 [BTC Listings] | Booting up http server | 0.0.0.0:80
btc_listings | 2020/04/08 16:22:48 [BTC Listings] | Starting periodic rating update routine
```


## Running without Docker-Compose

As the requirements require the capability to execute the server with: 

```bash
docker build -t test-server .
docker run --rm -p 80:80 test-server
```

For this to work, do be aware that, said setup must have the following:

- A boot up PostGreSQL database running (as the server would check and panic if not available)
- Token and configurations variables fed into environment using either `.env` file or plain environment variables.

I intentionally, added the [docker-compose.local.yml](./docker-compose.local.yml) to ease this needs so that testing
said application would be easier.

My aplogies if these requirements complicates things, but the generally idea was to not bake in these values right into
the docker image itself.

## Dependencies Setup
To setup locally, ensure to first download all modules for project with:

```bash
go mod download
```

### Database Setup 

The project utilizes docker to boot up a sample PostgreSQL database for development
and testing, so ensure to have docker and docker compose setup, see [Docker Setup for Mac](https://docs.docker.com/docker-for-mac/).

To setup database for production/development, and testing, simply execute:

```
make up-db
```

This will boot up postgre with docker-compose, setting up appropriate tables
for both production/development and testing.

If you prefer to do setup on a different database, not connected to docker-compose, then 
consider using the [Migration Script](./migrations/setup_db.sh), which will setup database and 
tables for both development/production and testing as necessary. It however requires specific 
environment variables to be available to it, see below:


```bash
# user for connecting to said database
$POSTGRES_USER

# password for connecting to said database
$POSTGRES_PASSWORD

# database to connect to with credentials
POSTGRES_DB
```

### How to run the test suite

Project comes with tests, and the database tests require postgres to be up and
running, see **Database Setup** section.

```
make test
```

