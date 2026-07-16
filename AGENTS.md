# AGENTS.md

## Learned User Preferences

- Prefer gobds packet-level block/stall (claims, pickups, other hot actions) so work never reaches BDS Script API; extend claim proxy rather than adding BEH hot-path handlers for the same denial.
- Open PR review threads: fix the issue or reply why intentional, then resolve — do not leave a pile of unanswered CodeRabbit/review comments.

## Learned Workspace Facts

- Claim proxy loads policy from `policy/claim_policy.v1.json` and can deny claim-related actions before they hit BDS; floor-only Y deny is intentional (not a bug).
- Traffic/metrics keys may use raw XUID as the ops identifier; do not "fix" that to a hashed form without an explicit product decision.
- bds-manager can build this repo from local source (`D:\bedrock-server\gobds`) into `bds-manager/gobds/gobds.exe` for the TESTING/Windows flow.
