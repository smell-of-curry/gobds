package cmd

// EngineResponseCommandType ...
type EngineResponseCommandType string

const (
	// EngineResponseCommandTypeLiteral represents a literal command type
	EngineResponseCommandTypeLiteral EngineResponseCommandType = "literal"
	// EngineResponseCommandTypeString represents a string command parameter type
	EngineResponseCommandTypeString EngineResponseCommandType = "string"
	// EngineResponseCommandTypeInt represents an integer command parameter type
	EngineResponseCommandTypeInt EngineResponseCommandType = "int"
	// EngineResponseCommandTypeFloat represents a float command parameter type
	EngineResponseCommandTypeFloat EngineResponseCommandType = "float"
	// EngineResponseCommandTypeLocation represents a location command parameter type
	EngineResponseCommandTypeLocation EngineResponseCommandType = "location"
	// EngineResponseCommandTypeBoolean represents a boolean command parameter type
	EngineResponseCommandTypeBoolean EngineResponseCommandType = "boolean"
	// EngineResponseCommandTypePlayer represents a player command parameter type
	EngineResponseCommandTypePlayer EngineResponseCommandType = "player"
	// EngineResponseCommandTypeTarget represents a target command parameter type
	EngineResponseCommandTypeTarget EngineResponseCommandType = "target"
	// EngineResponseCommandTypeArray represents an array command parameter type
	EngineResponseCommandTypeArray EngineResponseCommandType = "array"
	// EngineResponseCommandTypeDuration represents a duration command parameter type
	EngineResponseCommandTypeDuration EngineResponseCommandType = "duration"
	// EngineResponseCommandTypePlayerName represents a player name command parameter type
	EngineResponseCommandTypePlayerName EngineResponseCommandType = "playerName"
)

// EngineResponseCommand ...
type EngineResponseCommand struct {
	BaseCommand       string                       `json:"baseCommand"`
	Name              string                       `json:"name"`
	Description       string                       `json:"description"`
	Aliases           []string                     `json:"aliases,omitempty"`
	Type              EngineResponseCommandType    `json:"type"`
	AllowedTypeValues []string                     `json:"allowedTypeValues,omitempty"`
	Children          []EngineResponseCommandChild `json:"children"`
	CanBeCalled       bool                         `json:"canBeCalled"`
	RequiresOp        bool                         `json:"requiresOp"`
}

// EngineResponseCommandChild ...
type EngineResponseCommandChild struct {
	EngineResponseCommand
	Parent string `json:"parent"`
	Depth  int    `json:"depth"`
}
