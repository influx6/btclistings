build:
	docker build -t btc_listing_server .

run-dev:
	docker-compose up

run-prod:
	docker run -it --rm -p 80:80  btc_listing_server

test: unit

unit:
	go test -v ./pkg/...
