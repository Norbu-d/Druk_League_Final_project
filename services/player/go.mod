module github.com/drukleague/platform/services/player

go 1.24

require github.com/drukleague/platform/packages/servicekit v0.0.0

require (
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.9.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/net v0.60.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
)

replace github.com/drukleague/platform/packages/servicekit => ../../packages/servicekit

require (
	github.com/drukleague/platform/packages/contracts v0.0.0
	google.golang.org/grpc v1.83.1
	google.golang.org/protobuf v1.36.5
)

replace github.com/drukleague/platform/packages/contracts => ../../packages/contracts
