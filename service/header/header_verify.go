package header

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tendermint/tendermint/light"
)

// TrustingPeriod is period through which we can trust a header's validators set.
//
// Should be significantly less than the unbonding period (e.g. unbonding
// period = 3 weeks, trusting period = 2 weeks).
//
// More specifically, trusting period + time needed to check headers + time
// needed to report and punish misbehavior should be less than the unbonding
// period.
// TODO(@Wondertan): We should request it from the network's state params
//  or listen for network params changes to always have a topical value.
var TrustingPeriod = 168 * time.Hour

// IsExpired checks if header is expired against trusting period.
func (eh *ExtendedHeader) IsExpired() bool {
	expirationTime := eh.Time.Add(TrustingPeriod)
	return !expirationTime.After(time.Now())
}

// VerifyNonAdjacent validates non-adjacent untrusted header against trusted 'eh'.
func (eh *ExtendedHeader) VerifyNonAdjacent(untrst *ExtendedHeader) error {
	if untrst.ChainID != eh.ChainID {
		return &VerifyError{
			fmt.Errorf(
				"header belongs to another chain %q, not %q", untrst.ChainID, eh.ChainID,
			),
		}
	}

	if !untrst.Time.After(eh.Time) {
		return &VerifyError{
			fmt.Errorf("expected new header time %v to be after old header time %v", untrst.Time, eh.Time),
		}
	}

	now := time.Now()
	if !untrst.Time.Before(now) {
		return &VerifyError{
			fmt.Errorf("new header has a time from the future %v (now: %v)", untrst.Time, now),
		}
	}

	// Ensure that untrusted commit has enough of trusted commit's power.
	err := eh.ValidatorSet.VerifyCommitLightTrusting(eh.ChainID, untrst.Commit, light.DefaultTrustLevel)
	if err != nil {
		return &VerifyError{err}
	}

	return nil
}

// VerifyAdjacent validates adjacent untrusted header against trusted 'eh'.
func (eh *ExtendedHeader) VerifyAdjacent(untrst *ExtendedHeader) error {
	if untrst.Height != eh.Height+1 {
		return ErrNonAdjacent
	}

	if untrst.ChainID != eh.ChainID {
		return &VerifyError{
			fmt.Errorf("header belongs to another chain %q, not %q", untrst.ChainID, eh.ChainID),
		}
	}

	if !untrst.Time.After(eh.Time) {
		return &VerifyError{
			fmt.Errorf("expected new header time %v to be after old header time %v", untrst.Time, eh.Time),
		}
	}

	now := time.Now()
	if !untrst.Time.Before(now) {
		return &VerifyError{
			fmt.Errorf("new header has a time from the future %v (now: %v)", untrst.Time, now),
		}
	}

	// Check the validator hashes are the same
	if !bytes.Equal(untrst.ValidatorsHash, eh.NextValidatorsHash) {
		return &VerifyError{
			fmt.Errorf("expected old header next validators (%X) to match those from new header (%X)",
				eh.NextValidatorsHash,
				untrst.ValidatorsHash,
			),
		}
	}

	return nil
}

// VerifyError is thrown on during VerifyAdjacent and VerifyNonAdjacent if verification fails.
type VerifyError struct {
	Reason error
}

func (vr *VerifyError) Error() string {
	return fmt.Sprintf("header: verify: %s", vr.Reason.Error())
}
