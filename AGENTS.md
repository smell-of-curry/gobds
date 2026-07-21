# AGENTS.md

## Learned User Preferences

- Prefer gobds packet-level block/stall (claims, pickups, other hot actions) so work never reaches BDS Script API; extend claim proxy rather than adding BEH hot-path handlers for the same denial.
- Open PR review threads: fix the issue or reply why intentional, then resolve — do not leave a pile of unanswered CodeRabbit/review comments.

## Learned Workspace Facts

- Claim proxy loads policy from `policy/claim_policy.v1.json` and can deny claim-related actions before they hit BDS; floor-only Y deny is intentional (not a bug).
- Traffic/metrics keys may use raw XUID as the ops identifier; do not "fix" that to a hashed form without an explicit product decision.
- bds-manager can build this repo from local source (`D:\bedrock-server\gobds`) into `bds-manager/gobds/gobds.exe` for the TESTING/Windows flow.
- gobds terminates resource-pack delivery: clients get packs from the proxy's config URL list (`DownloadResourcePack=false`), so BDS/world packs never reach clients. Load URL packs via `readURLPack()` in `gobds/user_config.go` (download + `resource.Read`), NOT `resource.ReadURL` — `ReadURL` records the source URL as the pack's `downloadURL`, which the gophertunnel listener advertises as `TexturePackInfo.DownloadURL`, making clients self-download over HTTPS. That CDN path only fires on a pack cache miss (same UUID, new version) and fails on GitHub release asset URLs: 302 redirect, plus `manifest.json` at the zip root — the client-side CDN loader requires it inside a subfolder (per `resource.ReadURL` docs), while server-side compilation (`resource.Read`/`ReadPath`) accepts the root layout → players stuck on "Loading resource packs" or a half-applied pack. Empty `DownloadURL` serves packs over RakNet chunks, matching the hub (dragonfly serves unpacked pack dirs, no DownloadURL).
- Hub/downstream pack version skew is routine: bds-manager's `ProxyManager.generateConfig` fetches the LATEST pokebedrock-res GitHub release on every proxy start, while the hub only updates its pack on hub restart — pack delivery must tolerate the hub and downstream servers shipping different pack versions.
