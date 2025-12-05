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
	builder := newCommandBuilder(pkt)
	commands := cmd.Commands()
	for _, c := range commands {
		aliasesIndex := builder.processAliases(c)
		overloads := builder.processParams(c)

		builder.pkt.Commands = append(builder.pkt.Commands, protocol.Command{
			Name:          c.Name(),
			Description:   c.Description(),
			AliasesOffset: aliasesIndex,
			Overloads:     overloads,
		})
	}
	return builder.pkt
}

type commandBuilder struct {
	pkt                *packet.AvailableCommands
	enumIndices        map[string]uint32
	enumValueIndices   map[string]uint
	dynamicEnumIndices map[string]uint32
	suffixIndices      map[string]uint32
}

func newCommandBuilder(pkt *packet.AvailableCommands) *commandBuilder {
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
	return &commandBuilder{
		pkt:                pkt,
		enumIndices:        enumIndices,
		enumValueIndices:   enumValueIndices,
		dynamicEnumIndices: dynamicEnumIndices,
		suffixIndices:      suffixIndices,
	}
}

func (b *commandBuilder) processAliases(c cmd.Command) uint32 {
	if len(c.Aliases()) == 0 {
		return math.MaxUint32
	}

	aliasEnumType := c.Name() + "Aliases"
	if idx, ok := b.enumIndices[aliasEnumType]; ok {
		return idx
	}

	aliasesIndex := uint32(len(b.pkt.Enums))
	b.enumIndices[aliasEnumType] = aliasesIndex
	var valIdxs []uint
	for _, opt := range c.Aliases() {
		if vi, ok := b.enumValueIndices[opt]; ok {
			valIdxs = append(valIdxs, vi)
		} else {
			vi = uint(len(b.pkt.EnumValues))
			b.enumValueIndices[opt] = vi
			b.pkt.EnumValues = append(b.pkt.EnumValues, opt)
			valIdxs = append(valIdxs, vi)
		}
	}
	b.pkt.Enums = append(b.pkt.Enums, protocol.CommandEnum{
		Type:         aliasEnumType,
		ValueIndices: valIdxs,
	})
	return aliasesIndex
}

func (b *commandBuilder) processParams(c cmd.Command) []protocol.CommandOverload {
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
				t = b.handleEnum(t, enumDef)
			}
			if paramInfo.Suffix != "" {
				t = b.handleSuffix(t, paramInfo.Suffix)
			}
			overloads[i].Parameters = append(overloads[i].Parameters, protocol.CommandParameter{
				Name:     paramInfo.Name,
				Type:     t,
				Optional: paramInfo.Optional,
				Options:  opt,
			})
		}
	}
	return overloads
}

func (b *commandBuilder) handleEnum(t uint32, enumDef commandEnum) uint32 {
	if !enumDef.Dynamic {
		var enumIdx uint32
		if idx, ok := b.enumIndices[enumDef.Type]; ok {
			enumIdx = idx
		} else {
			enumIdx = uint32(len(b.pkt.Enums))
			b.enumIndices[enumDef.Type] = enumIdx
			var valIdxs []uint
			for _, optStr := range enumDef.Options {
				if vi, ok := b.enumValueIndices[optStr]; ok {
					valIdxs = append(valIdxs, vi)
				} else {
					vi = uint(len(b.pkt.EnumValues))
					b.enumValueIndices[optStr] = vi
					b.pkt.EnumValues = append(b.pkt.EnumValues, optStr)
					valIdxs = append(valIdxs, vi)
				}
			}
			b.pkt.Enums = append(b.pkt.Enums, protocol.CommandEnum{
				Type:         enumDef.Type,
				ValueIndices: valIdxs,
			})
		}
		t |= protocol.CommandArgEnum | enumIdx
	} else {
		var dynIdx uint32
		if idx, ok := b.dynamicEnumIndices[enumDef.Type]; ok {
			dynIdx = idx
		} else {
			dynIdx = uint32(len(b.pkt.DynamicEnums))
			b.dynamicEnumIndices[enumDef.Type] = dynIdx
			b.pkt.DynamicEnums = append(b.pkt.DynamicEnums, protocol.DynamicEnum{
				Type:   enumDef.Type,
				Values: enumDef.Options,
			})
		}
		t |= protocol.CommandArgSoftEnum | dynIdx
	}
	return t
}

func (b *commandBuilder) handleSuffix(t uint32, suffix string) uint32 {
	var suffIdx uint32
	if idx, ok := b.suffixIndices[suffix]; ok {
		suffIdx = idx
	} else {
		suffIdx = uint32(len(b.pkt.Suffixes))
		b.suffixIndices[suffix] = suffIdx
		b.pkt.Suffixes = append(b.pkt.Suffixes, suffix)
	}
	t |= protocol.CommandArgSuffixed | suffIdx
	return t
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
