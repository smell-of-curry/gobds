# Structure of NameTags Syntax 🏷️
TODO: Allow this system to be expanded to other syntax

## Overview

GoBDS provides automatic entity name tag translation functionality that dynamically displays entity names in the player's chosen language. This feature bypasses current Scripting API limitations by intercepting and modifying name tag packets before they reach the client.

## Features

- **Dynamic Language Translation** 🌐  
  Automatically translates entity names based on the player's selected language.

- **Nickname Preservation** ✏️  
  Properly handles and preserves nicknames without translating them.

- **Owner Information Display** 👤  
  Maintains ownership information in name tags (player name or trainer name).

- **Multi-line Format Support** 📝  
  Preserves the structure of complex name tags with level information and other details.

- **Translation Caching** ⚡  
  Caches translations for better performance.

## Name Tag Structure

Entity name tags follow specific formats that are preserved during translation:

### Wild Pokémon
```
§l[Pokémon Name]§r\n§eLvl [Level]
```
Example:
```
§lTaillow\n§eLvl 21§r
```

### Owned Pokémon
```
§l[Pokémon Name] §eLvl [Level]\n[Owner Name]'s§r
§l§n§r[Nickname] §eLvl [Level]\n[Owner Name]'s§r
```
Examples:
```
§lImpidimp §eLvl 25\nOwner not found§r
§lCharmander §eLvl 100\nSmell of curry§r
§lCharmander §eLvl 100\nSteve§r
§l§n§rMy Custom Name §eLvl 100\nSteve§r
```

### Trainer-owned Pokémon
```
§l[Pokémon Name] §eLvl [Level]\nTrainer: [Trainer Name]§r
§l§n§r[Nickname] §eLvl [Level]\nTrainer: [Trainer Name]§r
```
Examples:
```
§lCharmander §eLvl 13\nTrainer: BullGate§r
§lCharmander §eLvl 13\nTrainer: Trainer not found§r
§l§n§rMy Cool Custom Name §eLvl 13\nTrainer: Bull Dog§r
```

## Implementation Details

### Language Selection

The system automatically detects the player's preferred language from their client settings:
- If the language is supported in the resource pack, translations use that language
- If not supported, defaults to `en_US`

### Nickname Handling

Nicknames are identified by a special prefix (`§n§r`) and are preserved without translation.

### Translation Process

1. Entity packets are intercepted by the proxy
2. If the entity is a Pokémon (has prefix "pokemon:"), the name tag is processed
3. The system retrieves translations from the resource pack's language files
4. The original name is replaced with the translated version while preserving format
5. Modified packets are sent to the client

## Integration with Resource Packs

This system relies on properly formatted resource packs with language files containing entity name translations. Translations should follow the format:

```
"item.spawn_egg.entity.[entity_type_id].name": "[Translated Name]"
```

## Example

A Charmander owned by player "Steve" would be displayed as:
- English: `§lCharmander §eLvl 5\nSteve's§r`
- Spanish: `§lCharmander §eLvl 5\nSteve's§r`
- Japanese: `§lヒトカゲ §eLvl 5\nSteve's§r`

This dynamic translation happens in real-time without any additional configuration from server administrators.