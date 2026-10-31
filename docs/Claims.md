# Claims Overview 🏰

**GoBDS** includes a flexible claim system (adapted from the [vanilla-proxy](https://github.com/smell-of-curry/vanilla-proxy)) that allows you to define areas in the world where specific players (or everyone) can build, break, or interact with blocks and entities. This system helps protect builds, resources, and custom areas from unwanted access.

Below is a breakdown of how the claim logic works, including data structures, dimension management, and packet interception for enforcing claim rules.

## How claims are enforced

1. **Load.** Each server with `[Network.Servers.ClaimService]` polls `URL` on
   `PollInterval`. The body is a JSON array of `{data, _key}` rows. The proxy keeps the latest
   successful fetch as an immutable snapshot. Readers take the snapshot without
   a lock.
2. **Age.** A snapshot older than `MaxSnapshotAge` is not used. The action is
   passed to BDS. A panic in the claim path fails open the same way.
3. **Decide.** Break, place, and interact packets are matched against the
   snapshot. The owner XUID and `trusts` are allowed. An owner of `*` is an
   admin claim: most interactions stay allowed, and decoration hits (armor
   stand, painting, cushion) stay denied. Feature flags on the claim
   (`mineable`, `blockPlaceable`, `blockIntractable`, `entityIntractable`,
   `entityHurt`, `dropItems`) narrow that further.
4. **Drop.** A denied action is removed from the packet. BDS never sees it.
   `pickupItems` is stored for parity with the behavior pack and is not
   enforced: pickup is decided on the server.
5. **Draw.** `DenyRenderingEnabled` rewrites loaded subchunks so protected
   ground renders as deny blocks.

## Payload

Rows match `claim.PlayerClaim`. JSON names are what the HTTP endpoint returns.

```go
type PlayerClaim struct {
    ID           string    `json:"claimId"`
    OwnerXUID    string    `json:"playerXUID"`
    Location     Location  `json:"location"`
    Features     []Feature `json:"features"`
    TrustedXUIDS []string  `json:"trusts"`
}
```

| Field | JSON | |
| --- | --- | --- |
| `ID` | `claimId` | Unique id |
| `OwnerXUID` | `playerXUID` | Owner. `*` is an admin claim |
| `Location` | `location` | Dimension and the XZ box |
| `Features` | `features` | Extra allow rules inside the box |
| `TrustedXUIDS` | `trusts` | Other XUIDs with the owner's access |

```go
type Location struct {
    Dimension string  `json:"dimension"`
    Pos1      Vector2 `json:"pos1"`
    Pos2      Vector2 `json:"pos2"`
}

type Feature struct {
    Type          string   `json:"type"`
    FromLocation  *Vector3 `json:"fromLocation,omitempty"`
    ToLocation    *Vector3 `json:"toLocation,omitempty"`
    BlockTypeIDs  []string `json:"blockTypeIds,omitempty"`
    EntityTypeIDs []string `json:"entityTypeIds,omitempty"`
    ItemTypeIDs   []string `json:"itemTypeIds,omitempty"`
}
```

`type` is one of `mineable`, `blockPlaceable`, `blockIntractable`,
`entityIntractable`, `entityHurt`, `dropItems`, `pickupItems`. The two
`Intractable` spellings are the stored values.

`Pos1` / `Pos2` are the XZ corners. A feature's `fromLocation` / `toLocation`
add a Y range when the rule is not the whole column.