package cmd

// EngineResponseCommandType ...
type EngineResponseCommandType string

const (
	EngineResponseCommandTypeLiteral    EngineResponseCommandType = "literal"
	EngineResponseCommandTypeString     EngineResponseCommandType = "string"
	EngineResponseCommandTypeInt        EngineResponseCommandType = "int"
	EngineResponseCommandTypeFloat      EngineResponseCommandType = "float"
	EngineResponseCommandTypeLocation   EngineResponseCommandType = "location"
	EngineResponseCommandTypeBoolean    EngineResponseCommandType = "boolean"
	EngineResponseCommandTypePlayer     EngineResponseCommandType = "player"
	EngineResponseCommandTypeTarget     EngineResponseCommandType = "target"
	EngineResponseCommandTypeArray      EngineResponseCommandType = "array"
	EngineResponseCommandTypeDuration   EngineResponseCommandType = "duration"
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
