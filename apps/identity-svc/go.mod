module github.com/aureum/identity-svc

go 1.25.0

require (
	github.com/Nerzal/gocloak/v13 v13.9.0
	github.com/aureum/pkg v0.0.0-00010101000000-000000000000
	github.com/go-chi/chi/v5 v5.1.0
	github.com/jackc/pgx/v5 v5.7.2
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/redis/go-redis/v9 v9.7.0
	google.golang.org/grpc v1.80.0
)

require (
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-resty/resty/v2 v2.7.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.16 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pquerna/otp v1.4.0 // indirect
	github.com/segmentio/kafka-go v0.4.47 // indirect
	github.com/segmentio/ksuid v1.0.4 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/aureum/pkg => ../../pkg
	github.com/aureum/proto => ../../proto
)
