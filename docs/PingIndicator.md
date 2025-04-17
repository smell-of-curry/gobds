# Ping Indicator

The Ping Indicator feature displays a player's current ping (latency) directly in their view as a title message, making it easy for players to monitor their connection quality while playing.

## How It Works

The ping indicator:
- Updates every 20 ticks (approximately 1 second)
- Shows the player's current ping in milliseconds
- Color-codes the ping value based on connection quality:
  - **§a** (Green): Excellent connection (<20ms)
  - **§e** (Yellow): Good connection (<50ms) 
  - **§6** (Gold): Fair connection (<100ms)
  - **§c** (Red): Poor connection (<200ms)
  - **§4** (Dark Red): Very poor connection (≥200ms)

## Technical Implementation

The system uses the `SetTitle` packet with a special prefix (`&_playerPing:`) that can be detected and rendered by a resource pack:

```go
titlePk := &packet.SetTitle{
    ActionType: packet.TitleActionSetTitle,
    Text:       "&_playerPing:Current Ping: " + pingStatus + formattedPing,
}
```

This prefix allows resource packs to identify and display the ping indicator in a customized way, separate from regular title messages.

## Resource Pack Integration

To properly display the ping indicator, your resource pack should:

1. Detect the `&_playerPing:` prefix in title messages
2. Strip the prefix and render the ping information in a designated UI location
3. (Optional) Apply custom styling or icons based on the ping quality

An example integration of this in a resource pack can be found [here](https://github.com/smell-of-curry/pokebedrock-res/blob/main/ui/phud/playerPing.json)

## Configuration

The ping indicator is enabled by default. Configuration options may be added in future versions to:
- Adjust update frequency
- Customize color thresholds
- Change display format
- Toggle the feature on/off

## Benefits

- Players can monitor their connection quality in real-time
- Helps players troubleshoot performance issues
- Provides immediate feedback when connection quality changes
- Enhances the overall gameplay experience with minimal performance impact