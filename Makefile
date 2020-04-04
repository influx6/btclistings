start:
	docker run -it btc_list

test: unit

unit:
	go test -v ./pkg/...
