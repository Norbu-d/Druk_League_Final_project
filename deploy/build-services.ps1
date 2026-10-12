$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
Push-Location $projectRoot
try {
    docker build --tag drukleague-service-builder:local --file services/service-builder/Dockerfile .
    if ($LASTEXITCODE -ne 0) { throw "Shared service builder image build failed." }

    Push-Location (Join-Path $projectRoot "deploy")
    try {
        $services = @(
            "api", "auth", "user", "team", "player", "tournament", "registration",
            "match", "score", "fixture", "standings", "venue", "official",
            "dzongkhag", "notification", "media", "report", "search", "analytics",
            "audit-log", "configuration"
        )
        docker compose --env-file .env --parallel 2 build @services
        if ($LASTEXITCODE -ne 0) { throw "Service image build failed." }
    }
    finally {
        Pop-Location
    }
}
finally {
    Pop-Location
}