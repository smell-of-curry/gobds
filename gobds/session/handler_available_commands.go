package session

import (
	"math"
	"slices"
	"sync"

	"github.com/go-gl/mathgl/mgl64"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/cmd"
)

// AvailableCommandsHandler ...
type AvailableCommandsHandler struct {
	cache sync.Map
}

// disabledCommands ...
var disabledCommands = []string{"me", "tell", "w", "msg"}

// Handle ...
func (h *AvailableCommandsHandler) Handle(_ *Session, pk packet.Packet, _ *Context) error {
	pkt := pk.(*packet.AvailableCommands)

	h.cache.Clear()
	for _, c := range pkt.Commands {
		if !slices.Contains(disabledCommands, c.Name) {
			h.cache.Store(c.Name, c)
		}
	}

	*pkt = *h.appendCustomCommands(pkt)
	return nil
}

// appendCustomCommands ...
func (h *AvailableCommandsHandler) appendCustomCommands(pkt *packet.AvailableCommands) *packet.AvailableCommands {
	enumIndices := make(map[string]uint32, len(pkt.Enums))
	for i, e := range pkt.Enums {
		enumIndices[e.Type] = uint32(i)
	}
	enumValueIndices := make(map[string]uint, len(pkt.EnumValues))
	for i, val := range pkt.EnumValues {
		enumValueIndices[val] = uint(i)
	}
	dynamicEnumIndices := make(map[string]uint32, len(pkt.DynamicEnums))
	for i, de := range pkt.DynamicEnums {
		dynamicEnumIndices[de.Type] = uint32(i)
	}
	suffixIndices := make(map[string]uint32, len(pkt.Suffixes))
	for i, s := range pkt.Suffixes {
		suffixIndices[s] = uint32(i)
	}

	commands := cmd.Commands()
	for _, c := range commands {
		aliasesIndex := uint32(math.MaxUint32)
		if len(c.Aliases()) > 0 {
			aliasEnumType := c.Name() + "Aliases"
			if idx, ok := enumIndices[aliasEnumType]; ok {
				aliasesIndex = idx
			} else {
				aliasesIndex = uint32(len(pkt.Enums))
				enumIndices[aliasEnumType] = aliasesIndex
				var valIdxs []uint
				for _, opt := range c.Aliases() {
					if vi, ok := enumValueIndices[opt]; ok {
						valIdxs = append(valIdxs, vi)
					} else {
						vi = uint(len(pkt.EnumValues))
						enumValueIndices[opt] = vi
						pkt.EnumValues = append(pkt.EnumValues, opt)
						valIdxs = append(valIdxs, vi)
					}
				}
				pkt.Enums = append(pkt.Enums, protocol.CommandEnum{
					Type:         aliasEnumType,
					ValueIndices: valIdxs,
				})
			}
		}

		params := c.Params()
		overloads := make([]protocol.CommandOverload, len(params))
		for i, paramList := range params {
			for _, paramInfo := range paramList {
				t, enumDef := valueToParamType(paramInfo)
				t |= protocol.CommandArgValid
				opt := byte(0)
				if _, isBool := paramInfo.Value.(bool); isBool {
					opt |= protocol.ParamOptionCollapseEnum
				}
				if enumDef.Type != "" {
					if !enumDef.Dynamic {
						var enumIdx uint32
						if idx, ok := enumIndices[enumDef.Type]; ok {
							enumIdx = idx
						} else {
							enumIdx = uint32(len(pkt.Enums))
							enumIndices[enumDef.Type] = enumIdx
							var valIdxs []uint
							for _, optStr := range enumDef.Options {
								if vi, ok := enumValueIndices[optStr]; ok {
									valIdxs = append(valIdxs, vi)
								} else {
									vi = uint(len(pkt.EnumValues))
									enumValueIndices[optStr] = vi
									pkt.EnumValues = append(pkt.EnumValues, optStr)
									valIdxs = append(valIdxs, vi)
								}
							}
							pkt.Enums = append(pkt.Enums, protocol.CommandEnum{
								Type:         enumDef.Type,
								ValueIndices: valIdxs,
							})
						}
						t |= protocol.CommandArgEnum | enumIdx
					} else {
						var dynIdx uint32
						if idx, ok := dynamicEnumIndices[enumDef.Type]; ok {
							dynIdx = idx
						} else {
							dynIdx = uint32(len(pkt.DynamicEnums))
							dynamicEnumIndices[enumDef.Type] = dynIdx
							pkt.DynamicEnums = append(pkt.DynamicEnums, protocol.DynamicEnum{
								Type:   enumDef.Type,
								Values: enumDef.Options,
							})
						}
						t |= protocol.CommandArgSoftEnum | dynIdx
					}
				}
				if paramInfo.Suffix != "" {
					var suffIdx uint32
					if idx, ok := suffixIndices[paramInfo.Suffix]; ok {
						suffIdx = idx
					} else {
						suffIdx = uint32(len(pkt.Suffixes))
						suffixIndices[paramInfo.Suffix] = suffIdx
						pkt.Suffixes = append(pkt.Suffixes, paramInfo.Suffix)
					}
					t |= protocol.CommandArgSuffixed | suffIdx
				}
				overloads[i].Parameters = append(overloads[i].Parameters, protocol.CommandParameter{
					Name:     paramInfo.Name,
					Type:     t,
					Optional: paramInfo.Optional,
					Options:  opt,
				})
			}
		}

		pkt.Commands = append(pkt.Commands, protocol.Command{
			Name:          c.Name(),
			Description:   c.Description(),
			AliasesOffset: aliasesIndex,
			Overloads:     overloads,
		})
	}
	return pkt
}

// commandEnum ...
type commandEnum struct {
	Type    string
	Options []string
	Dynamic bool
}

// valueToParamType finds the command argument type of the value passed and returns it, in addition to creating
// an enum if applicable.
func valueToParamType(i cmd.ParamInfo) (t uint32, enum commandEnum) {
	switch v := i.Value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return protocol.CommandArgTypeInt, enum
	case float32, float64:
		return protocol.CommandArgTypeFloat, enum
	case string:
		if v == "target" {
			return protocol.CommandArgTypeTarget, enum
		}
		return protocol.CommandArgTypeString, enum
	case cmd.Varargs:
		return protocol.CommandArgTypeRawText, enum
	case bool:
		return 0, commandEnum{
			Type:    "bool",
			Options: []string{"true", "false"},
		}
	case mgl64.Vec3:
		return protocol.CommandArgTypePosition, enum
	case cmd.SubCommand:
		return 0, commandEnum{
			Type:    "SubCommand" + i.Name,
			Options: []string{i.Name},
		}
	}
	if enum, ok := i.Value.(cmd.Enum); ok {
		return 0, commandEnum{
			Type:    enum.Type(),
			Options: enum.Options(),
			Dynamic: true,
		}
	}
	return protocol.CommandArgTypeValue, enum
}
