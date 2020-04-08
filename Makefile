COIN_API_TOKEN?=
POSTGRES_USER?=postgres
POSTGRES_PASSWORD?=starcraft
POSTGRES_HOST=localhost:5432
DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}/btc_listings"
DATABASE_TEST_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}/btc_listings_test"

build:
	docker build -t btc_listing_server .

prod:
	docker run -it --rm -p 80:80  btc_listing_server

db-up:
	docker-compose up

db-down:
	docker-compose down

up:
	docker-compose -f docker-compose.local.yml up

down:
	docker-compose -f docker-compose.local.yml down

run:
	env COIN_API_TOKEN=${COIN_API_TOKEN} DATABASE_URL=${DATABASE_URL} HOST="localhost" PORT="3040" go run cmd/btclistings/main.go

test: unit

unit:
	env DATABASE_URL=${DATABASE_TEST_URL} go test -v ./pkg/...
