.PHONY: mqstart mqstop mqlog server client

mqstart:
	./rabbit.sh start

mqstop:
	./rabbit.sh stop

mqlog:
	./rabbit.sh logs

server:
	go run ./cmd/server

client:
	go run ./cmd/client
