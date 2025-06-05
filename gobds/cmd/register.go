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
			Register(New(c.Name, c.Description, c.Aliases, nil))
			continue
		}

		var maxDepth int
		for _, child := range c.Children {
			if child.Depth > maxDepth {
				maxDepth = child.Depth
			}
		}

		params := make([][]ParamInfo, maxDepth)
		for _, child := range c.Children {
			depthIndex := child.Depth - 1
			if depthIndex < 0 || depthIndex >= maxDepth {
				continue
			}
			var value any
			if len(child.AllowedTypeValues) > 0 {
				value = staticEnum{
					typ:     fmt.Sprintf("%sEnum", child.Name),
					options: child.AllowedTypeValues,
				}
			} else {
				switch child.Type {
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
				Name:     child.Name,
				Value:    value,
				Optional: false,
			}
			params[depthIndex] = append(params[depthIndex], p)
		}
		Register(New(c.Name, c.Description, c.Aliases, params))
	}
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
