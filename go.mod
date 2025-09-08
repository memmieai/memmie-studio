module github.com/memmieai/memmie-studio

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/jmoiron/sqlx v1.3.5
	github.com/lib/pq v1.10.9
	github.com/memmieai/memmie-common v0.0.0
	github.com/nats-io/nats.go v1.31.0
	github.com/redis/go-redis/v9 v9.5.1
	github.com/stretchr/testify v1.9.0
	go.temporal.io/sdk v1.26.1
	go.uber.org/zap v1.27.0
)

replace github.com/memmieai/memmie-common => ../memmie-common