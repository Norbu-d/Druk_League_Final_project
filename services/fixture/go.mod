module github.com/drukleague/platform/services/fixture

go 1.26.0

require github.com/drukleague/platform/packages/servicekit v0.0.0

require (
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.9.2 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/nats-io/nkeys v0.4.9 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/net v0.60.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
)

replace github.com/drukleague/platform/packages/servicekit => ../../packages/servicekit

replace github.com/drukleague/platform/packages/contracts => ../../packages/contracts

require (
	github.com/drukleague/platform/packages/contracts v0.0.0
	github.com/drukleague/platform/packages/eventkit v0.0.0
	github.com/nats-io/nats.go v1.39.1
	google.golang.org/grpc v1.83.2
	google.golang.org/protobuf v1.36.11
)

replace github.com/drukleague/platform/packages/eventkit => ../../packages/eventkit
