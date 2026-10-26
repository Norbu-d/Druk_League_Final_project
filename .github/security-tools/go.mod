module github.com/drukleague/platform/.github/security-tools

go 1.26.0

tool (
	github.com/securego/gosec/v2/cmd/gosec
	golang.org/x/vuln/cmd/govulncheck
)

require (
	github.com/securego/gosec/v2 v2.30.0
	golang.org/x/vuln v1.1.4
)
