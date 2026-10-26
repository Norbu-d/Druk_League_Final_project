package main

import (
	"context"
	"database/sql"
	"log"
	"strconv"
	"time"

	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/user/internal/adapters/grpc"
	"github.com/drukleague/platform/services/user/internal/adapters/hasher"
	"github.com/drukleague/platform/services/user/internal/adapters/postgres"
	"github.com/drukleague/platform/services/user/internal/app"
	"google.golang.org/grpc"
)

func main() {
	cost, _ := strconv.Atoi(servicekit.Env("BCRYPT_COST", "12"))
	register := func(server *grpc.Server, db *sql.DB) error {
		service := app.New(postgres.New(db), hasher.New(cost))
		// The first super admin comes from configuration; later password changes in .env do not
		// overwrite an existing admin (same behaviour as the monolith).
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		created, err := service.Bootstrap(ctx, servicekit.Env("BOOTSTRAP_ADMIN_USER", "admin"), servicekit.Env("BOOTSTRAP_ADMIN_PASSWORD", "drukleague-dev"))
		if err != nil {
			return err
		}
		if created {
			log.Printf("bootstrap super admin created")
		}
		v1.RegisterUserServiceServer(server, grpcadapter.New(service))
		return nil
	}
	if err := servicekit.Run("user-service", register); err != nil {
		log.Fatal(err)
	}
}
