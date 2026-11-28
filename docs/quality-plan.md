# SWE303 / SWE302 — Quality Plan

Version 1.0, 28 Nov 2026. This plan sets the project's quality objectives, how each one is checked, and the measured result. Detailed test cases are in [swe302-test-plan.md](swe302-test-plan.md); executed results are in [test-execution-report.md](test-execution-report.md).

## 1. Quality objectives

| # | Objective | Target | How it is checked | Result on 28 Nov 2026 |
|---|---|---|---|---|
| Q1 | Unit-test coverage per service | ≥ 80% statements in each module | `scripts/coverage.sh 80` in CI (fails the build below 80%) | **Met:** all 25 measured modules 80.4–100% (lowest: media 80.4%) |
| Q2 | Contract compatibility | No breaking change to published protobufs | `buf lint`, `buf build`, `buf breaking` in CI | Met in CI |
| Q3 | No known critical vulnerabilities | 0 CRITICAL in images; 0 reachable Go vulnerabilities | Trivy on every image; govulncheck and gosec | Met in CI (images, Go modules) |
| Q4 | Authorisation correct for every role | Every write is refused for the wrong role | Unit tests per service and BFF; real-stack acceptance (36 checks) | **Met** after D-010 was fixed (stale player image) |
| Q5 | Result integrity | Only the assigned official finalises; final results are immutable | Score tests; live check: second finalise → 409 | Met |
| Q6 | Critical journeys work end to end | Guest, sign-up, admin, manager, official flows | Playwright suite; acceptance scripts | See the test execution report |
| Q7 | Web security baseline | No high-risk alerts | OWASP ZAP baseline scan; CSP and security headers in nginx | See the test execution report |
| Q8 | Performance | Login and public standings p95 within the k6 thresholds | k6 job in CI | Met in CI |
| Q9 | No open high-severity defects at submission | 0 open High | [defect-log.md](defect-log.md) | See the defect log |

## 2. Standards

- **Code:** gofmt, go vet, the hexagonal layout per service (`internal/domain`, `app`, `ports`, `adapters`); a doc comment on each exported type and on non-obvious functions.
- **Contracts:** versioned packages under `proto/drukleague`; Buf lint rules.
- **Security:** least privilege (each service checks the caller's role); no secrets in Git (`deploy/.env` is ignored); distroless non-root images; HttpOnly SameSite=Strict cookies; a strict CSP.
- **Web:** server responses are escaped in the UI (`esc()`); API paths are restricted by an allowlist (`role-permissions.js`); CSV exports are formula-safe.

## 3. Quality assurance activities

| Activity | When | Who |
|---|---|---|
| Unit tests with the code | Every change | Author |
| Pull-request review plus required CI checks | Every merge to `development` | Reviewer and CI |
| Integration tests against PostgreSQL (`TEST_DATABASE_URL`) and embedded JetStream | Every CI run | CI |
| Contract checks (Buf) | Every CI run | CI |
| Security scans (gosec, govulncheck, Trivy) | Every CI run | CI |
| Performance smoke (k6) and audit end-to-end test | Every CI run | CI |
| Browser end-to-end tests (Playwright) | Before each demo and release | QA owner |
| Real-stack acceptance on a test-only database | After each feature milestone | Feature owner |
| OWASP ZAP baseline scan | Before the release | QA owner |
| Defect triage | Sprint review | Team |

## 4. Quality control: CI gates

From `.github/workflows/go-ci.yml`, a merge is blocked unless all of these pass:
1. Go formatting, vet and build for every module, including `functions/*`.
2. The per-module coverage gate (80%).
3. Buf lint, build and breaking checks.
4. gosec and govulncheck.
5. Building every service and function image, plus a Trivy CRITICAL scan.
6. Frontend role-permission tests.
7. k6 login and public-standings performance.
8. API admin action → audit log end-to-end test.

## 5. Defect management

Defects are logged in [defect-log.md](defect-log.md) with severity (P1 blocks the demo or risks unauthorised or corrupt results; P2 affects a major workflow; P3 is cosmetic), reproduction steps and the fixing revision. A defect is closed only after its reproduction passes on a recorded revision.

## 6. Quality records

- `progress.md`: dated evidence for each acceptance run.
- [test-execution-report.md](test-execution-report.md): results per level, with dates.
- CI run history on GitHub for each pull request.
