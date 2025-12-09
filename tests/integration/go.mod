module github.com/prohmpiriya/booking-rush-10k-rps/tests/integration

go 1.24.0

toolchain go1.24.11

require github.com/prohmpiriya/booking-rush-10k-rps/pkg v0.0.0

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/redis/go-redis/v9 v9.17.2 // indirect
)

replace github.com/prohmpiriya/booking-rush-10k-rps/pkg => ../../pkg
