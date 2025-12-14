package session

import (
	"fmt"
	"hash/fnv"
	"maps"
	"sort"

	"github.com/df-mc/dragonfly/server/world"
)

// hash ...
// Credit: https://discord.com/channels/623638955262345216/637335508166377513/1347856008038449222
func hash(b world.Block) uint32 {
	name, properties := b.EncodeBlock()
	l := int16(len(name))
	data := []byte{
		10,   // TAG_Compound
		0, 0, // length
		8,    // TAG_String
		4, 0, // length
		110, 97, 109, 101, // "name"
		byte(l), byte(l >> 8), // length
	}
	data = append(data, []byte(name)...)
	data = append(data,
		10,   // TAG_Compound
		6, 0, // length
		115, 116, 97, 116, 101, 115, // "states"
	)

	keys := maps.Keys(properties)
	sort.Strings(keys)
	for _, key := range keys {
		value := properties[key]
		var tagType byte
		var tagData []byte
		switch v := value.(type) {
		case byte:
			tagType = 1
			tagData = []byte{v}
		case bool:
			tagType = 1
			if v {
				tagData = []byte{1}
			} else {
				tagData = []byte{0}
			}
		case int32:
			tagType = 3
			tagData = []byte{
				byte(v),
				byte(v >> 8),
				byte(v >> 16),
				byte(v >> 24),
			}
		case string:
			tagType = 8
			l = int16(len(v))
			tagData = append([]byte{byte(l), byte(l >> 8)}, []byte(v)...)
		default:
			panic(fmt.Sprintf("unknown state type: key=%v, value=%v, type=%T", key, value, value))
		}
		l = int16(len(key))
		data = append(data, tagType, byte(l), byte(l>>8))
		data = append(data, []byte(key)...)
		data = append(data, tagData...)
	}
	data = append(data, 0, 0) // TAG_End, TAG_End

	fnvHash := fnv.New32a()
	_, _ = fnvHash.Write(data)
	rid := fnvHash.Sum32()
	fnvHash.Reset()
	return rid
}
