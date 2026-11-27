package servicekit

import (
	"context"
	"log"
	"os"
	"regexp"
	"time"

	configurationv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/configuration/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var configurationSettingName = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// configurationSettings is the allowlist of non-secret settings services may
// read from ConfigurationService. Credentials and storage paths stay local.
var configurationSettings = map[string]bool{
	"GRPC_ADDR": true, "METRICS_ADDR": true, "NATS_URL": true,
	"ACCESS_TOKEN_TTL": true, "REFRESH_TOKEN_TTL": true, "BCRYPT_COST": true,
	"TEAM_SERVICE_ADDR": true, "USER_SERVICE_ADDR": true, "TOURNAMENT_SERVICE_ADDR": true,
	"MATCH_SERVICE_ADDR": true, "PLAYER_SERVICE_ADDR": true, "REGISTRATION_SERVICE_ADDR": true,
	"VENUE_SERVICE_ADDR": true, "STANDINGS_SERVICE_ADDR": true, "FIXTURE_SERVICE_ADDR": true,
	"DZONGKHAG_SERVICE_ADDR": true, "AUTH_SERVICE_ADDR": true, "SCORE_SERVICE_ADDR": true,
	"OFFICIAL_SERVICE_ADDR": true, "REPORT_PUBLIC_BASE_URL": true, "REPORT_HTTP_ADDR": true,
	"MEDIA_HTTP_ADDR": true, "API_ADDR": true, "CORS_ORIGIN": true, "BFF_ADDR": true,
}

// LoadConfiguration overlays explicitly allowlisted, non-secret settings from
// ConfigurationService onto environment defaults before a service starts. A
// missing/unavailable Config service leaves the environment untouched.
func LoadConfiguration(serviceName string, settingNames ...string) {
	address := os.Getenv("CONFIGURATION_SERVICE_ADDR")
	if address == "" || serviceName == "" || serviceName == "configuration-service" || len(settingNames) == 0 {
		return
	}

	conn, err := Dial(address, ClientOptions{
		Auth:        ServiceIdentity,
		Tokens:      NewTokenSource(serviceName, Secret(), 0),
		Timeout:     750 * time.Millisecond,
		MaxAttempts: 1,
	})
	if err != nil {
		log.Printf("configuration service unavailable; using environment settings: %v", err)
		return
	}
	defer conn.Close()
	client := configurationv1.NewConfigurationServiceClient(conn)

	for _, name := range settingNames {
		if !configurationSettingName.MatchString(name) || !configurationSettings[name] {
			log.Printf("ignoring non-allowlisted configuration setting %q", name)
			continue
		}
		key := "service." + serviceName + "." + name
		var response *configurationv1.GetConfigurationResponse
		for attempt := 0; attempt < 3; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			response, err = client.GetConfiguration(ctx, &configurationv1.GetConfigurationRequest{Key: key})
			cancel()
			if err == nil || status.Code(err) == codes.NotFound {
				break
			}
			if (status.Code(err) != codes.Unavailable && status.Code(err) != codes.DeadlineExceeded) || attempt == 2 {
				break
			}
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
		}
		if status.Code(err) == codes.NotFound {
			continue
		}
		if err != nil {
			log.Printf("configuration lookup failed for service %s; using environment settings: %v", serviceName, err)
			return
		}
		if response == nil || response.GetEntry() == nil || response.GetEntry().GetKey() != key || response.GetEntry().GetValue() == "" {
			continue
		}
		if err := os.Setenv(name, response.GetEntry().GetValue()); err != nil {
			log.Printf("could not apply centralized setting %s for service %s: %v", name, serviceName, err)
			continue
		}
		log.Printf("loaded centralized setting %s for service %s", name, serviceName)
	}
}
