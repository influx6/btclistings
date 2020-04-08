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

```bash
make up
```

Beyond requiring docker-compose and docker (which most do have), with that simple command,
the necessary server and database would be booted up ready for testing. Do note that due to 
not setting up `a wait till db is ready before running server` type of script, docker-compose will
attempt to restart the server until it can successfully connect to the database, as the database
setups and executes migration scripts to prepare the db and table.

## Running without Docker-Compose

As the requirements require the capability to execute the server with: 

```bash
docker build -t test-server .
docker run --rm -p 80:80 test-server
```

For this to work, do be aware that, said setup must have the following:

- A boot up PostGreSQL database running (as the server would check and panic if not available)
- Token and configuration feed into environment using environment variables or as prescribed environment files

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

### How to run project locally

Once docker and database setup as describe in previous section is completed, we need to setup an `.env` file to host environment variables used by the application.

*This is required for both development, production and when executing tests, if you prefer feeding these one by one using `env VAR=VAL` then this will work as well.*

Add the following to an '.env' file:

```bash
# set database url for use by docker image
DATABASE_URL=postgres://postgres:starcraft@db:5432/btc_listings

# set coin api token 
COIN_API_TOKEN=26******-**********-*********-5D5D

# set host and port for go server
HOST=localhost

PORT=80
```

To run server and db, simply execute the following:

```
make up
```

You should be ready to hit the API with requests.

### How to run the test suite

Project comes with tests, and the database tests require postgres to be up and
running, see **Database Setup** section.

```
make test
```

