# HTTP smoke test of the bff, driving the same routes the web app uses.
param([string]$Base = 'http://localhost:8080/api', [string]$WebBase = 'http://localhost:4173/api')
$ErrorActionPreference = 'Stop'
$failures = 0
function Check($name, $ok, $detail) { if ($ok) { "PASS  $name" } else { $script:failures++; "FAIL  $name $detail" } }
function Call($method, $path, $token, $body) {
  $headers = @{}; if ($token) { $headers.Authorization = "Bearer $token" }
  $args = @{ Method = $method; Uri = "$Base$path"; Headers = $headers; ContentType = 'application/json' }
  if ($null -ne $body) { $args.Body = ($body | ConvertTo-Json -Compress) }
  Invoke-RestMethod @args
}
function List($path, $token) { foreach ($item in (Call GET $path $token $null)) { $item } }
function Code($method, $path, $token, $body) {
  try { Call $method $path $token $body | Out-Null; 200 } catch { [int]$_.Exception.Response.StatusCode }
}

$line = Select-String -Path D:\druk-league\deploy\.env -Pattern '^BOOTSTRAP_ADMIN_PASSWORD=' | Select-Object -First 1
$adminPassword = ($line.Line -split '=', 2)[1].Trim().Trim('"').Trim("'")
$run = (Get-Date).ToUniversalTime().ToString('MMddHHmmss')

Check 'no token is 401' ((Code GET '/teams' $null $null) -eq 401)
$login = Call POST '/auth/login' $null @{ username = 'admin'; password = $adminPassword }
Check 'admin login returns token, refresh token and user' ($login.token -and $login.refresh_token -and $login.user.role -eq 'super_admin')
$admin = $login.token
$renewed = Call POST '/auth/refresh' $null @{ refresh_token = $login.refresh_token }
Check 'refresh returns a new access token' ($renewed.token -and $renewed.refresh_token -ne $login.refresh_token)
$admin = $renewed.token

# A different start date per run, so repeated runs do not book the same venue at the same kickoff.
$seconds = [long][math]::Floor(((Get-Date).ToUniversalTime() - (Get-Date '2026-01-01')).TotalSeconds)
$startDate = (Get-Date '2027-01-01').AddDays($seconds % 9000).ToString('yyyy-MM-dd')
$cup = Call POST '/tournaments' $admin @{ name = "BFF Cup $run"; sport = 'football'; season = '2026'; venue = ''; starts_at = $startDate; dzongkhag = 'thimphu' }
Check 'create tournament (date input, dzongkhag canonicalised)' ($cup.id -and $cup.dzongkhag -eq 'Thimphu' -and $cup.status -eq 'registration_open')
$list = @(List '/tournaments' $admin)
Check 'list tournaments includes it' (@($list | Where-Object id -eq $cup.id).Count -eq 1)
$homeTeam = Call POST '/teams' $admin @{ name = "Paro$run"; short_code = 'P'; dzongkhag = 'Paro'; tournament_id = $cup.id }
$awayTeam = Call POST '/teams' $admin @{ name = "Haa United $run"; short_code = 'HU'; dzongkhag = 'Haa'; tournament_id = $cup.id }
Check 'create teams (one-letter code widened)' ($homeTeam.short_code -eq 'PAR' -and $awayTeam.short_code -eq 'HU')
$fixtures = Call POST "/tournaments/$($cup.id)/fixtures" $admin $null
Check 'generate fixtures through Fixture service' ($fixtures.fixtures_created -eq 1) $fixtures
# A second tournament in the same dzongkhag on the same day: Fixture must move around the first one's booking.
$rival = Call POST '/tournaments' $admin @{ name = "BFF Rival Cup $run"; sport = 'football'; season = '2026'; venue = ''; starts_at = $startDate; dzongkhag = 'Thimphu' }
Call POST '/teams' $admin @{ name = "Punakha Rival $run"; short_code = 'PR'; dzongkhag = 'Punakha'; tournament_id = $rival.id } | Out-Null
Call POST '/teams' $admin @{ name = "Wangdue Rival $run"; short_code = 'WR'; dzongkhag = 'Wangdue Phodrang'; tournament_id = $rival.id } | Out-Null
$rivalFixtures = Call POST "/tournaments/$($rival.id)/fixtures" $admin $null
$first = @(List "/matches?tournament_id=$($cup.id)" $admin)[0]
$second = @(List "/matches?tournament_id=$($rival.id)" $admin)[0]
$moved = ($first.venue -ne $second.venue) -or ([math]::Abs(([datetime]$first.kickoff - [datetime]$second.kickoff).TotalHours) -ge 2)
Check 'same-day tournament in the same dzongkhag is rescheduled, not refused' ($rivalFixtures.fixtures_created -eq 1 -and $moved) "$($first.venue) $($first.kickoff) / $($second.venue) $($second.kickoff)"

$matches = @(List "/matches?tournament_id=$($cup.id)" $admin)
$match = $matches[0]
Check 'match row has team names from Team service' ($matches.Count -eq 1 -and @($match.home, $match.away) -contains "Paro$run" -and $match.status -eq 'scheduled') ($matches | ConvertTo-Json -Compress)

$officialName = "bff-official-$run"
$official = Call POST '/users' $admin @{ username = $officialName; password = 'official-password-1'; role = 'match_official' }
$officialToken = (Call POST '/auth/login' $null @{ username = $officialName; password = 'official-password-1' }).token
Check 'admin cannot score' ((Code PATCH "/matches/$($match.id)/live" $admin @{ home = 1; away = 0 }) -eq 403)
Call PATCH "/matches/$($match.id)/official" $admin @{ official_id = $official.id } | Out-Null
Call PATCH "/matches/$($match.id)/live" $officialToken @{ home = 1; away = 0 } | Out-Null
$liveRow = @(List "/matches?tournament_id=$($cup.id)" $admin)[0]
Check 'live score shows as live 1-0' ($liveRow.status -eq 'live' -and $liveRow.live_home -eq 1) ($liveRow | ConvertTo-Json -Compress)
$final = Call POST "/matches/$($match.id)/finalize" $officialToken @{ home = 2; away = 1 }
Check 'finalise uses the submitted 2-1, not the saved 1-0' ($final.home -eq 2 -and $final.away -eq 1) ($final | ConvertTo-Json -Compress)
Check 'finalised result cannot be changed' ((Code PATCH "/matches/$($match.id)/live" $officialToken @{ home = 0; away = 0 }) -eq 409)

$table = @()
foreach ($i in 1..20) { $table = @(List "/standings?tournament_id=$($cup.id)" $admin); if ($table.Count -eq 2) { break }; Start-Sleep -Seconds 1 }
Check 'standings updated from MatchFinalized event' ($table.Count -eq 2 -and $table[0].points -eq 3 -and $table[0].team) ($table | ConvertTo-Json -Compress)
$doneRow = @(List "/matches?tournament_id=$($cup.id)" $admin)[0]
Check 'match row shows completed 2-1' ($doneRow.status -eq 'completed' -and $doneRow.result_home -eq 2 -and $doneRow.result_away -eq 1) ($doneRow | ConvertTo-Json -Compress)

$viaNginx = Invoke-RestMethod -Uri "$WebBase/teams" -Headers @{ Authorization = "Bearer $admin" }
$proxied = @(foreach ($t in $viaNginx) { $t })
Check 'nginx /api proxies to the bff' ($proxied.Count -ge 2) $proxied.Count
Call POST '/auth/logout' $officialToken $null | Out-Null
Check 'logout revokes the session' ((Code POST '/auth/refresh' $null @{ refresh_token = 'x' }) -eq 401)

"`n$failures checks failed"
