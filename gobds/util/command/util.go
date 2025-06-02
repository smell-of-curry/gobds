package command

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// MergeAvailableCommands ...
func MergeAvailableCommands(pk1, pk2 packet.AvailableCommands) packet.AvailableCommands {
	mergedPacket := packet.AvailableCommands{}

	EnumValues, pk1EnumValuesMap, pk2EnumValuesMap := mergeUniqueStrings(pk1.EnumValues, pk2.EnumValues)
	mergedPacket.EnumValues = EnumValues

	ChainedSubcommandValues, pk1ChainedMap, pk2ChainedMap := mergeUniqueStrings(pk1.ChainedSubcommandValues, pk2.ChainedSubcommandValues)
	mergedPacket.ChainedSubcommandValues = ChainedSubcommandValues

	Enums, pk1EnumMap, pk2EnumMap := mergeUniqueEnums(pk1.Enums, pk2.Enums, pk1EnumValuesMap, pk2EnumValuesMap)
	mergedPacket.Enums = Enums

	Suffixes, _, _ := mergeUniqueStrings(pk1.Suffixes, pk2.Suffixes)
	mergedPacket.Suffixes = Suffixes

	mergedPacket.ChainedSubcommands = mergeChainedSubcommands(pk1.ChainedSubcommands, pk2.ChainedSubcommands, pk1ChainedMap, pk2ChainedMap)

	DynamicEnums, _, _ := mergeUniqueDynamicEnumsWithIndexMaps(pk1.DynamicEnums, pk2.DynamicEnums)
	mergedPacket.DynamicEnums = DynamicEnums

	mergedPacket.Commands = mergeUniqueCommands(pk1.Commands, pk2.Commands, pk1ChainedMap, pk2ChainedMap, pk1EnumMap, pk2EnumMap)
	mergedPacket.Constraints = mergeUniqueConstraints(pk1.Constraints, pk2.Constraints, pk1EnumValuesMap, pk2EnumValuesMap)
	return mergedPacket
}

// mergeUniqueStrings ...
func mergeUniqueStrings(slice1, slice2 []string) ([]string, map[uint]uint, map[uint]uint) {
	uniqueMap := make(map[string]uint)
	indexMap1 := make(map[uint]uint)
	indexMap2 := make(map[uint]uint)
	uniqueSlice := make([]string, 0)

	for i, item := range slice1 {
		if idx, exists := uniqueMap[item]; exists {
			indexMap1[uint(i)] = idx
		} else {
			newIdx := uint(len(uniqueSlice))
			uniqueMap[item] = newIdx
			indexMap1[uint(i)] = newIdx
			uniqueSlice = append(uniqueSlice, item)
		}
	}

	for i, item := range slice2 {
		if idx, exists := uniqueMap[item]; exists {
			indexMap2[uint(i)] = idx
		} else {
			newIdx := uint(len(uniqueSlice))
			uniqueMap[item] = newIdx
			indexMap2[uint(i)] = newIdx
			uniqueSlice = append(uniqueSlice, item)
		}
	}
	return uniqueSlice, indexMap1, indexMap2
}

// mergeUniqueEnums ...
func mergeUniqueEnums(slice1, slice2 []protocol.CommandEnum, enumValuesIndexMap1, enumValuesIndexMap2 map[uint]uint) ([]protocol.CommandEnum, map[uint]uint, map[uint]uint) {
	uniqueMap := make(map[string]uint)
	indexMap1 := make(map[uint]uint)
	indexMap2 := make(map[uint]uint)
	uniqueSlice := make([]protocol.CommandEnum, 0)

	for i, item := range slice1 {
		item = updateEnumIndices(item, enumValuesIndexMap1)
		if idx, exists := uniqueMap[item.Type]; exists {
			indexMap1[uint(i)] = idx
			uniqueSlice[idx] = mergeEnums(uniqueSlice[idx], item)
		} else {
			newIdx := uint(len(uniqueSlice))
			uniqueMap[item.Type] = newIdx
			indexMap1[uint(i)] = newIdx
			uniqueSlice = append(uniqueSlice, item)
		}
	}

	for i, item := range slice2 {
		item = updateEnumIndices(item, enumValuesIndexMap2)
		if idx, exists := uniqueMap[item.Type]; exists {
			indexMap2[uint(i)] = idx
			uniqueSlice[idx] = mergeEnums(uniqueSlice[idx], item)
		} else {
			newIdx := uint(len(uniqueSlice))
			uniqueMap[item.Type] = newIdx
			indexMap2[uint(i)] = newIdx
			uniqueSlice = append(uniqueSlice, item)
		}
	}
	return uniqueSlice, indexMap1, indexMap2
}

// mergeEnums ...
func mergeEnums(enum1, enum2 protocol.CommandEnum) protocol.CommandEnum {
	uniqueIndices := make(map[uint]struct{})
	for _, idx := range enum1.ValueIndices {
		uniqueIndices[idx] = struct{}{}
	}
	for _, idx := range enum2.ValueIndices {
		uniqueIndices[idx] = struct{}{}
	}
	mergedIndices := make([]uint, 0, len(uniqueIndices))
	for idx := range uniqueIndices {
		mergedIndices = append(mergedIndices, idx)
	}
	enum1.ValueIndices = mergedIndices
	return enum1
}

// updateEnumIndices ...
func updateEnumIndices(enum protocol.CommandEnum, indexMap map[uint]uint) protocol.CommandEnum {
	updatedEnum := enum
	updatedEnum.ValueIndices = make([]uint, len(enum.ValueIndices))
	for i, idx := range enum.ValueIndices {
		updatedEnum.ValueIndices[i] = indexMap[idx]
	}
	return updatedEnum
}

// mergeChainedSubcommands ...
func mergeChainedSubcommands(slice1, slice2 []protocol.ChainedSubcommand, indexMap1, indexMap2 map[uint]uint) []protocol.ChainedSubcommand {
	mergedSlice := make([]protocol.ChainedSubcommand, 0)
	for _, item := range slice1 {
		mergedSlice = append(mergedSlice, updateChainedSubcommandIndices(item, indexMap1))
	}
	for _, item := range slice2 {
		mergedSlice = append(mergedSlice, updateChainedSubcommandIndices(item, indexMap2))
	}
	return mergedSlice
}

// updateChainedSubcommandIndices ...
func updateChainedSubcommandIndices(chained protocol.ChainedSubcommand, indexMap map[uint]uint) protocol.ChainedSubcommand {
	updatedChained := chained
	updatedChained.Values = make([]protocol.ChainedSubcommandValue, len(chained.Values))
	for i, val := range chained.Values {
		val.Index = uint16(indexMap[uint(val.Index)])
		updatedChained.Values[i] = val
	}
	return updatedChained
}

// mergeUniqueCommands ...
func mergeUniqueCommands(slice1, slice2 []protocol.Command, chainedCommandsIndexMap1, chainedCommandsIndexMap2, enumsIndexMap1, enumsIndexMap2 map[uint]uint) []protocol.Command {
	uniqueMap := make(map[string]protocol.Command)
	for _, item := range slice1 {
		uniqueMap[item.Name] = updateCommandIndices(item, chainedCommandsIndexMap1, enumsIndexMap1)
	}
	for _, item := range slice2 {
		if _, exists := uniqueMap[item.Name]; exists {
			continue // skip duplicates
		} else {
			uniqueMap[item.Name] = updateCommandIndices(item, chainedCommandsIndexMap2, enumsIndexMap2)
		}
	}
	uniqueSlice := make([]protocol.Command, 0, len(uniqueMap))
	for _, item := range uniqueMap {
		uniqueSlice = append(uniqueSlice, item)
	}
	return uniqueSlice
}

// updateCommandIndices ...
func updateCommandIndices(cmd protocol.Command, chainedCommandsIndexMap, enumsIndexMap map[uint]uint) protocol.Command {
	updatedCmd := cmd
	updatedCmd.ChainedSubcommandOffsets = make([]uint16, len(cmd.ChainedSubcommandOffsets))
	for i, offset := range cmd.ChainedSubcommandOffsets {
		updatedCmd.ChainedSubcommandOffsets[i] = uint16(chainedCommandsIndexMap[uint(offset)])
	}
	if updatedCmd.AliasesOffset != ^uint32(0) {
		updatedCmd.AliasesOffset = uint32(enumsIndexMap[uint(cmd.AliasesOffset)])
	}
	return updatedCmd
}

// mergeUniqueDynamicEnumsWithIndexMaps ...
func mergeUniqueDynamicEnumsWithIndexMaps(slice1, slice2 []protocol.DynamicEnum) ([]protocol.DynamicEnum, map[uint]uint, map[uint]uint) {
	uniqueMap := make(map[string]uint)
	indexMap1 := make(map[uint]uint)
	indexMap2 := make(map[uint]uint)
	uniqueSlice := make([]protocol.DynamicEnum, 0)

	for i, item := range slice1 {
		if idx, exists := uniqueMap[item.Type]; exists {
			indexMap1[uint(i)] = idx
			mergedValues, _, _ := mergeUniqueStrings(uniqueSlice[idx].Values, item.Values)
			uniqueSlice[idx].Values = mergedValues
		} else {
			newIdx := uint(len(uniqueSlice))
			uniqueMap[item.Type] = newIdx
			indexMap1[uint(i)] = newIdx
			uniqueSlice = append(uniqueSlice, item)
		}
	}

	for i, item := range slice2 {
		if idx, exists := uniqueMap[item.Type]; exists {
			indexMap2[uint(i)] = idx
			mergedValues, _, _ := mergeUniqueStrings(uniqueSlice[idx].Values, item.Values)
			uniqueSlice[idx].Values = mergedValues
		} else {
			newIdx := uint(len(uniqueSlice))
			uniqueMap[item.Type] = newIdx
			indexMap2[uint(i)] = newIdx
			uniqueSlice = append(uniqueSlice, item)
		}
	}
	return uniqueSlice, indexMap1, indexMap2
}

// mergeUniqueConstraints ...
func mergeUniqueConstraints(slice1, slice2 []protocol.CommandEnumConstraint, indexMap1, indexMap2 map[uint]uint) []protocol.CommandEnumConstraint {
	uniqueMap := make(map[uint32]protocol.CommandEnumConstraint)
	for _, item := range slice1 {
		uniqueMap[item.EnumIndex] = updateConstraintIndices(item, indexMap1)
	}
	for _, item := range slice2 {
		if existing, exists := uniqueMap[item.EnumIndex]; exists {
			uniqueMap[item.EnumIndex] = mergeConstraints(existing, updateConstraintIndices(item, indexMap2))
		} else {
			uniqueMap[item.EnumIndex] = updateConstraintIndices(item, indexMap2)
		}
	}
	uniqueSlice := make([]protocol.CommandEnumConstraint, 0, len(uniqueMap))
	for _, item := range uniqueMap {
		uniqueSlice = append(uniqueSlice, item)
	}
	return uniqueSlice
}

// mergeConstraints ...
func mergeConstraints(constraint1, constraint2 protocol.CommandEnumConstraint) protocol.CommandEnumConstraint {
	uniqueConstraints := make(map[byte]struct{})
	for _, c := range constraint1.Constraints {
		uniqueConstraints[c] = struct{}{}
	}
	for _, c := range constraint2.Constraints {
		uniqueConstraints[c] = struct{}{}
	}
	mergedConstraints := make([]byte, 0, len(uniqueConstraints))
	for c := range uniqueConstraints {
		mergedConstraints = append(mergedConstraints, c)
	}
	constraint1.Constraints = mergedConstraints
	return constraint1
}

// updateConstraintIndices ...
func updateConstraintIndices(constraint protocol.CommandEnumConstraint, indexMap map[uint]uint) protocol.CommandEnumConstraint {
	updatedConstraint := constraint
	updatedConstraint.EnumValueIndex = uint32(indexMap[uint(constraint.EnumValueIndex)])
	updatedConstraint.EnumIndex = uint32(indexMap[uint(constraint.EnumIndex)])
	return updatedConstraint
}
