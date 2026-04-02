package gobds

import "github.com/google/uuid"

// selfSignedIDNamespace is the fixed namespace for deriving deterministic
// SelfSignedIDs from XUIDs via UUID v5 (SHA-1). The same namespace is used
// in the bds-manager migration script to ensure parity.
var selfSignedIDNamespace = uuid.MustParse("a3e78ee7-823a-4cb5-9fe0-532b54ccc20d")

// selfSignedIDFromXUID returns a deterministic SelfSignedID for a given XUID.
// Every Xbox account gets a unique, stable ID regardless of which device
// connects, eliminating the shared-device collision bug.
func selfSignedIDFromXUID(xuid string) string {
	return uuid.NewSHA1(selfSignedIDNamespace, []byte(xuid)).String()
}
