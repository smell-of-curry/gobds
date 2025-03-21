# Claims Overview 🏰

**GoBDS** includes a flexible claim system (adapted from the [vanilla-proxy](https://github.com/smell-of-curry/vanilla-proxy)) that allows you to define areas in the world where specific players (or everyone) can build, break, or interact with blocks and entities. This system helps protect builds, resources, and custom areas from unwanted access.

Below is a breakdown of how the claim logic works, including data structures, dimension management, and packet interception for enforcing claim rules.

## How Claims Work

1. **Claims Storage & Loading**  
   Claims are stored in a database (using a helper function `utils.FetchDatabase`). On server startup, `FetchClaims()` loads all claims into an in-memory map called `RegisteredClaims`. (TODO: Might want to make this load better)

2. **Dimension Mapping**  
   Each claim is tied to a Minecraft dimension (Overworld, Nether, End). A helper function `ClaimDimensionToInt()` converts a dimension name (e.g., `minecraft:overworld`) to an integer used internally by Bedrock (0 for Overworld, 1 for Nether, 2 for End).

3. **Player Position Check**  
   When a player attempts to interact with a block or use an item, the proxy checks if the target position or the player’s location is inside a defined claim. If the claim is found, **permission checks** ensure that only the owner or trusted players can perform the requested action.

4. **Packet Interception**  
   Two primary packet handlers, `ClaimPlayerAuthInputHandler` and `ClaimInventoryTransactionHandler`, intercept player actions:
   - **`ClaimPlayerAuthInputHandler`**: Monitors block-related actions (placing, breaking, etc.).  
   - **`ClaimInventoryTransactionHandler`**: Monitors item usage (e.g., throwing projectiles, using potions, etc.).

5. **Permission Checks**  
   If a claim is owned by the player’s XUID or the XUID is in the claim’s `trusts` list, the action is allowed. Otherwise, the action is blocked, and an error message and sound is triggered.

## Data Structures

### `IPlayerClaim`

```go
type IPlayerClaim struct {
    ClaimId    string   `json:"claimId"`
    PlayerXUID string   `json:"playerXUID"`
    Location   Location `json:"location"`
    Trusts     []string `json:"trusts"`
}
```

| Field          | Description                                                               |
|----------------|---------------------------------------------------------------------------|
| **ClaimId**    | Unique identifier for the claim (e.g., UUID).                             |
| **PlayerXUID** | The XUID of the player who owns the claim. A special `*` can designate an "admin" claim open to all certain interactions. |
| **Location**   | Boundaries and dimension info. See `Location` below.                      |
| **Trusts**     | A list of other player XUIDs who share full permissions on this claim.     |

### `Location`

```go
type Location struct {
    Dimension string   `json:"dimension"`
    Pos1      VectorXZ `json:"pos1"`
    Pos2      VectorXZ `json:"pos2"`
}
```

| Field          | Description                                             |
|----------------|---------------------------------------------------------|
| **Dimension**  | String ID of the dimension (e.g., `minecraft:nether`).  |
| **Pos1**       | Lower corner (X,Z) of the claim boundary.               |
| **Pos2**       | Upper corner (X,Z) of the claim boundary.               |

### `VectorXZ`

```go
type VectorXZ struct {
    X float32 `json:"x"`
    Z float32 `json:"z"`
}
```

Represents a 2D coordinate on the XZ plane.