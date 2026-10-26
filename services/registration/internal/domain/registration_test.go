package domain

import "testing"

func TestValidStatus(t *testing.T) {
    for _, s := range []string{StatusPending, StatusApproved, StatusRejected} {
        if !ValidStatus(s) {
            t.Fatalf("%s should be valid", s)
        }
    }
    for _, s := range []string{"", "done", "APPROVED"} {
        if ValidStatus(s) {
            t.Fatalf("%q should be invalid", s)
        }
    }
}
