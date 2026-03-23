module github.com/dreamph/cachez/stores/redis

go 1.23.0

require (
	github.com/alicebob/miniredis/v2 v2.37.0
	github.com/dreamph/cachez v0.0.0
	github.com/go-redis/cache/v9 v9.0.0
	github.com/redis/go-redis/v9 v9.18.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/vmihailenco/go-tinylfu v0.2.2 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.4 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
)

replace github.com/dreamph/cachez => ../..
