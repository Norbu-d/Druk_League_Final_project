param(
    [string]$OutputDirectory = (Join-Path $env:LOCALAPPDATA 'DrukLeague\backups')
)

$ErrorActionPreference = 'Stop'
$composeFile = Join-Path (Split-Path $PSScriptRoot -Parent) 'deploy/docker-compose.yml'
$docker = (Get-Command docker -ErrorAction Stop).Source
$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'

New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
$outputPath = (Resolve-Path $OutputDirectory).Path
$volumeMountPath = $outputPath -replace '\\', '/'
$databaseBackup = Join-Path $outputPath "drukleague-databases-$stamp.sql"
$mediaBackupName = "drukleague-media-$stamp.tar.gz"

$configLines = & $docker compose -f $composeFile config --format json
if ($LASTEXITCODE -ne 0) { throw 'Could not read Compose configuration.' }
$composeConfig = ($configLines -join "`n") | ConvertFrom-Json
$mediaVolume = $composeConfig.volumes.'media-data'.name
if ([string]::IsNullOrWhiteSpace($mediaVolume)) { throw 'The Compose media-data volume is not configured.' }

$start = [System.Diagnostics.ProcessStartInfo]::new()
$start.FileName = $docker
$start.UseShellExecute = $false
$start.RedirectStandardOutput = $true
$start.RedirectStandardError = $true
@('compose', '-f', $composeFile, 'exec', '-T', 'postgres', 'sh', '-c', 'pg_dumpall -U "$POSTGRES_USER"') |
    ForEach-Object { [void]$start.ArgumentList.Add($_) }
$process = [System.Diagnostics.Process]::new()
$process.StartInfo = $start
if (-not $process.Start()) { throw 'Could not start the PostgreSQL backup command.' }
try {
    $stream = [System.IO.File]::Create($databaseBackup)
    try { $process.StandardOutput.BaseStream.CopyTo($stream) }
    finally { $stream.Dispose() }
    $process.WaitForExit()
    $stderr = $process.StandardError.ReadToEnd()
    if ($process.ExitCode -ne 0) {
        Remove-Item -LiteralPath $databaseBackup -Force -ErrorAction SilentlyContinue
        throw "Database backup failed (exit $($process.ExitCode)): $stderr"
    }
}
finally { $process.Dispose() }

$mediaCommand = "tar -czf /backup/$mediaBackupName -C /data ."
& $docker run --rm `
    --mount "type=volume,src=$mediaVolume,dst=/data,readonly" `
    --mount "type=bind,src=$volumeMountPath,dst=/backup" `
    alpine:3.22 sh -c $mediaCommand
if ($LASTEXITCODE -ne 0) { throw 'Media volume backup failed; the database dump was retained.' }

Write-Output "Database backup: $databaseBackup"
Write-Output "Media backup: $(Join-Path $outputPath $mediaBackupName)"
