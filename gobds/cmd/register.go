package cmd

import (
	"fmt"
	"sync"

	"github.com/go-gl/mathgl/mgl64"
)

// commands holds a list of registered commands indexed by their name.
var commands sync.Map

// Register registers a command with its name and all aliases that it has. Any command with the same name or
// aliases will be overwritten.
func Register(command Command) {
	commands.Store(command.name, command)
}

// staticEnum ...
type staticEnum struct {
	typ     string
	options []string
}

func (s staticEnum) Type() string      { return s.typ }
func (s staticEnum) Options() []string { return s.options }

// LoadFrom ...
func LoadFrom(commands map[string]EngineResponseCommand) {
	for _, c := range commands {
		if len(c.Children) == 0 {
			Register(New(c.Name, c.Description, c.Aliases, nil, c.RequiresOp))
			continue
		}

		// Build all possible command paths by following the nested structure
		overloads := buildCommandOverloads(c.Children)

		// If the root command is callable, add an empty overload (command can be called without parameters)
		if c.CanBeCalled {
			overloads = append([][]ParamInfo{{}}, overloads...)
		}

		// If no overloads were created and root command is not callable, create an empty command
		if len(overloads) == 0 {
			overloads = [][]ParamInfo{{}}
		}

		Register(New(c.Name, c.Description, c.Aliases, overloads, c.RequiresOp))
	}
}

// buildCommandOverloads creates separate parameter overloads for each complete command path
func buildCommandOverloads(children []EngineResponseCommandChild) [][]ParamInfo {
	var overloads [][]ParamInfo

	// For each direct child, build all possible paths
	for _, child := range children {
		paths := buildCommandPaths(child)
		for _, path := range paths {
			var params []ParamInfo
			for _, node := range path {
				var value any
				if len(node.AllowedTypeValues) > 0 {
					value = staticEnum{
						typ:     fmt.Sprintf("%sEnum", node.Name),
						options: node.AllowedTypeValues,
					}
				} else {
					switch node.Type {
					case EngineResponseCommandTypeLiteral:
						value = SubCommand{}
					case EngineResponseCommandTypeString, EngineResponseCommandTypePlayerName, EngineResponseCommandTypeDuration:
						value = ""
					case EngineResponseCommandTypeInt:
						value = int(0)
					case EngineResponseCommandTypeFloat:
						value = float64(0)
					case EngineResponseCommandTypeBoolean:
						value = false
					case EngineResponseCommandTypeLocation:
						value = mgl64.Vec3{0, 0, 0}
					case EngineResponseCommandTypePlayer, EngineResponseCommandTypeTarget:
						value = "target"
					default:
						value = ""
					}
				}
				p := ParamInfo{
					Name:     node.Name,
					Value:    value,
					Optional: false,
				}
				params = append(params, p)
			}
			if len(params) > 0 {
				overloads = append(overloads, params)
			}
		}
	}

	return overloads
}

// buildCommandPaths builds all possible paths starting from a given child by following nested structure
func buildCommandPaths(current EngineResponseCommandChild) [][]EngineResponseCommandChild {
	var allPaths [][]EngineResponseCommandChild

	// If this node is callable, include it as a complete path
	if current.CanBeCalled {
		allPaths = append(allPaths, []EngineResponseCommandChild{current})
	}

	// If this node has children, recursively build paths for each child
	if len(current.Children) > 0 {
		for _, child := range current.Children {
			childPaths := buildCommandPaths(child)
			for _, childPath := range childPaths {
				// Prepend current node to each child path
				fullPath := append([]EngineResponseCommandChild{current}, childPath...)
				allPaths = append(allPaths, fullPath)
			}
		}
	}

	return allPaths
}

// Commands returns a map of all registered commands indexed by the alias they were registered with.
func Commands() map[string]Command {
	cmd := make(map[string]Command)
	commands.Range(func(key, value any) bool {
		cmd[key.(string)] = value.(Command)
		return true
	})
	return cmd
}
