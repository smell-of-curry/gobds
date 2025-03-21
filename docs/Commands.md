# Commands Overview 📜

**GoBDS** (inspired by [vanilla-proxy](https://github.com/smell-of-curry/vanilla-proxy)) supports a dynamic command registration system that allows a downstream server (via the Scripting API or another mechanism) to send commands to the proxy, which are then relayed to Bedrock clients. This enables servers to:

- Dynamically register new commands on the fly  
- Provide command auto-completion, descriptions, aliases, and parameter typing  
- Merge custom commands with existing Bedrock Dedicated Server (BDS) commands  

Below is an explanation of the relevant structures, how they are interpreted by the proxy, and how these commands become available in-game.

## How Commands Are Received

1. **Text Packet Check**  
   When the Bedrock client sends or receives a `Text` packet of `TextTypeObject`, the proxy examines its content:
   ```go
   if dataPacket.TextType != packet.TextTypeObject {
       return true, dataPacket, nil
   }
   ```
2. **Custom Prefix**  
   We then look for a special prefix:  
   ```go
   if !strings.HasPrefix(message, "[PROXY_SYSTEM][COMMANDS]=") {
       // Not a command update packet
       return true, dataPacket, nil
   }
   ```
   This prefix (`[PROXY_SYSTEM][COMMANDS]=`) indicates that the message contains JSON data describing the commands to register.

3. **JSON Parsing**  
   The command data, stripped of the prefix, is then unmarshaled into a Go structure:
   ```go
   var commands map[string]IEngineResponseCommand
   err := json.Unmarshal([]byte(commandsRaw), &commands)
   ```
   Each key in the map typically matches the base command name.

4. **AvailableCommands Packet**  
   The parsed command data is converted to an official Bedrock `AvailableCommands` packet. This packet:
   - Enumerates all commands and their parameters
   - Assigns parameter types (int, float, string, bool, etc.)
   - Handles aliases
   - Manages enumerations (for example, listing possible array or boolean values)

5. **Merging With BDS Commands**  
   The newly generated commands are merged with any that BDS has already sent:
   ```go
   availableCommands = command.MergeAvailableCommands(availableCommands, bdsSentCommands)
   ```
   This ensures custom commands and native BDS commands coexist.

6. **Sending to the Client**  
   Finally, the proxy sends the `AvailableCommands` packet to the client:
   ```go
   player.DataPacket(&availableCommands)
   ```
   This triggers the in-game letting players see and use the new commands immediately.

## Data Structures

### `IEngineResponseCommand`
A high-level command definition with the following fields:

```go
type IEngineResponseCommand struct {
    BaseCommand       string   `json:"baseCommand"`
    Name              string   `json:"name"`
    Description       string   `json:"description"`
    Aliases           []string `json:"aliases,omitempty"`
    Type              string   `json:"type"`
    AllowedTypeValues []string `json:"allowedTypeValues,omitempty"`
    Children          []IEngineResponseCommandChild `json:"children"`
    CanBeCalled       bool     `json:"canBeCalled"`
    RequiresOp        bool     `json:"requiresOp"`
}
```

| Field             | Description                                                                 |
|-------------------|-----------------------------------------------------------------------------|
| **BaseCommand**   | (Optional usage detail) A raw base command reference.                       |
| **Name**          | The canonical name of the command (e.g., `teleport`).                        |
| **Description**   | A user-facing description.                                                   |
| **Aliases**       | List of alternative names for the command (e.g., `["tp"]`).                  |
| **Type**          | A string classification (often unused at the root level).                    |
| **AllowedTypeValues** | Potential values, used for enumerations or array-based parameters.       |
| **Children**      | An array of sub-commands/parameters that define usage.                       |
| **CanBeCalled**   | Indicates if the command can be executed.                                    |
| **RequiresOp**    | Whether the command needs OP-level permission.                               |

### `IEngineResponseCommandChild`
Sub-commands or parameters for a command:

```go
type IEngineResponseCommandChild struct {
    IEngineResponseCommand
    Parent string `json:"parent"`
    Depth  int    `json:"depth"`
}
```

| Field                 | Description                                                                     |
|-----------------------|---------------------------------------------------------------------------------|
| **IEngineResponseCommand** | Inherits the same fields as above (Name, Type, etc.).                      |
| **Parent**            | The parent command this child belongs to.                                       |
| **Depth**             | Nesting level for complex multi-parameter commands.                             |

---

## Parameter Type Mapping

Each child’s `Type` is mapped to a corresponding Bedrock parameter type, which influences auto-completion and validation. Common mappings:

| Custom Type   | Bedrock Arg Type             | Notes                                                           |
|---------------|------------------------------|-----------------------------------------------------------------|
| **literal**   | CommandArgTypeString (enum)  | Treated as a static literal (like sub-command keywords).         |
| **string**    | CommandArgTypeString         | Free-form text.                                                 |
| **int**       | CommandArgTypeInt            | Integer values only.                                            |
| **float**     | CommandArgTypeFloat          | Float/double values only.                                       |
| **location**  | CommandArgTypePosition       | Typically X, Y, Z coordinates.                                  |
| **boolean**   | Enum with `true/false`       | Collapsed boolean choice.                                       |
| **player**    | CommandArgTypeTarget         | Target selection (single or multiple players).                  |
| **target**    | CommandArgTypeTarget         | Alias for player-type targeting but can use entity refrence.                                |
| **array**     | Enum with `AllowedTypeValues`| Limited selection from a pre-defined list.                      |
| **duration**  | CommandArgTypeString         | Example: “1m30s” or “2h” — raw string requiring custom parsing. |
| **playerName**| CommandArgTypeString         | Free-form but intended for a player name.                       |

---

## Example JSON

Below is a simplified example of how a custom command might be sent to the proxy:

```json
{
  "teleport": {
    "baseCommand": "teleport",
    "name": "teleport",
    "description": "Teleport a player to a specific location or another player.",
    "aliases": ["tp"],
    "type": "root",
    "children": [
      {
        "baseCommand": "teleport",
        "name": "target",
        "description": "Player(s) to teleport.",
        "type": "player",
        "parent": "teleport",
        "depth": 1
      },
      {
        "baseCommand": "teleport",
        "name": "location",
        "description": "Coordinates to teleport to.",
        "type": "location",
        "parent": "teleport",
        "depth": 1
      }
    ],
    "canBeCalled": true,
    "requiresOp": false
  }
}
```

When this JSON (as a string) is prefixed with `[PROXY_SYSTEM][COMMANDS]=` and sent in a `TextTypeObject` packet, the proxy will:

1. Parse the JSON
2. Convert it into an `AvailableCommands` packet
3. Merge it with existing BDS commands
4. Send it to the player to enable auto-complete and usage

## Key Takeaways

1. **Dynamic Registration**: You can register commands at runtime by sending the JSON data through the custom prefix.
2. **Auto-Completion**: Bedrock’s client automatically respects the parameter types, descriptions, and enumerations set in the commands.
3. **Merging**: Custom commands are merged with default BDS commands for a seamless user experience.
4. **Extensibility**: Support for boolean, array, literal, and other specialized types gives you the flexibility to build complex commands.

## Tips & Best Practices

- **Ensure Unique Names**: If multiple commands share the same `name` and `aliases`, the client will only see one set of completions.
- **Use Children Wisely**: Complex commands with multiple parameters should be broken into logical children to benefit from auto-completion at each step.
- **Validate on Execution**: Bedrock auto-completion is helpful, but always validate parameters on the server side to prevent edge cases or exploits.
- **Keep JSON Small**: Sending huge command structures frequently can be inefficient. Send only relevant command updates or cache them on the server and trigger updates sparingly.