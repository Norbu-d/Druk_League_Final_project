package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	reportv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/report/v1"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	grpcadapter "github.com/drukleague/platform/services/report/internal/adapters/grpc"
	"github.com/drukleague/platform/services/report/internal/adapters/grpcclients"
	httpadapter "github.com/drukleague/platform/services/report/internal/adapters/http"
	"github.com/drukleague/platform/services/report/internal/adapters/postgres"
	"github.com/drukleague/platform/services/report/internal/app"
	"google.golang.org/grpc"
)

func main() {
	servicekit.LoadConfiguration("report-service",
		"GRPC_ADDR",
		"METRICS_ADDR",
		"STANDINGS_SERVICE_ADDR",
		"FIXTURE_SERVICE_ADDR",
		"REPORT_PUBLIC_BASE_URL",
		"REPORT_HTTP_ADDR",
	)
	serviceToken := servicekit.NewTokenSource("report-service", servicekit.Secret(), 0)
	standingsConn, err := servicekit.Dial(servicekit.Env("STANDINGS_SERVICE_ADDR", "standings:50051"), servicekit.ClientOptions{
		Auth: servicekit.ServiceIdentity, Tokens: serviceToken, Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer standingsConn.Close()
	fixturesConn, err := servicekit.Dial(servicekit.Env("FIXTURE_SERVICE_ADDR", "fixture:50051"), servicekit.ClientOptions{
		Auth: servicekit.ServiceIdentity, Tokens: serviceToken, Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer fixturesConn.Close()

	standings := grpcclients.NewStandings(standingsConn)
	fixtures := grpcclients.NewFixtures(fixturesConn)
	var downloadServer *http.Server
	var downloadListener net.Listener
	err = servicekit.Run("report-service", func(server *grpc.Server, db *sql.DB) error {
		service := app.New(postgres.New(db), standings, fixtures, servicekit.Env("REPORT_PUBLIC_BASE_URL", "/api/reports/files"))
		reportv1.RegisterReportServiceServer(server, grpcadapter.New(service))
		downloadServer = &http.Server{
			Addr: servicekit.Env("REPORT_HTTP_ADDR", ":8080"), Handler: httpadapter.New(service, servicekit.Secret()),
			ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second,
		}
		downloadListener, err = net.Listen("tcp", downloadServer.Addr)
		if err != nil {
			return err
		}
		go func() {
			if err := downloadServer.Serve(downloadListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("report download HTTP server: %v", err)
			}
		}()
		return nil
	})
	if downloadServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if shutdownErr := downloadServer.Shutdown(ctx); shutdownErr != nil {
			log.Printf("report download HTTP shutdown: %v", shutdownErr)
			_ = downloadListener.Close()
		}
		cancel()
	}
	if err != nil {
		log.Fatal(err)
	}
}
