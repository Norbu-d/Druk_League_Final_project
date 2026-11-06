package auditoutbox

import (
	"errors"
	"strings"
)

// Subject validates and normalizes an outbox subject before publishing it.
func Subject(subject string) (string, error) {
	// Rows written before the outbox began storing NATS subjects used this label.
	if subject == "MatchFinalized" {
		return "drukleague.match.finalized", nil
	}
	if subject == "drukleague.match.finalized" || subject == "drukleague.fixtures.generated" ||
		(strings.HasPrefix(subject, "drukleague.admin.") && !strings.ContainsAny(subject, "* >")) {
		return subject, nil
	}
	return "", errors.New("unsupported event subject")
}
