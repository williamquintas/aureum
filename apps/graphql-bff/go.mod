module github.com/aureum/graphql-bff

go 1.25.0

replace (
	github.com/aureum/pkg => ../../pkg
	github.com/aureum/proto => ../../proto
)

require (
	github.com/99designs/gqlgen v0.17.90
	github.com/go-chi/chi/v5 v5.1.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/vektah/gqlparser/v2 v2.5.33
	google.golang.org/grpc v1.80.0
)

require (
	github.com/agnivade/levenshtein v1.2.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/sosodev/duration v1.4.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260401024825-9d38bb4040a9 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
