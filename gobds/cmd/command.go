package cmd

// Command ...
type Command struct {
	name        string
	description string
	aliases     []string
	params      [][]ParamInfo
}

// New ...
func New(name, description string, aliases []string, params [][]ParamInfo) Command {
	return Command{
		name:        name,
		description: description,
		aliases:     aliases,
		params:      params,
	}
}

// Name returns the name of the command.
func (cmd Command) Name() string {
	return cmd.name
}

// Description returns the description of the command.
func (cmd Command) Description() string {
	return cmd.description
}

// Aliases returns a list of aliases for the command.
func (cmd Command) Aliases() []string {
	return cmd.aliases
}

// Params returns a list of all parameters.
func (cmd Command) Params() [][]ParamInfo {
	return cmd.params
}

// ParamInfo holds the information of a parameter.
type ParamInfo struct {
	Name     string
	Value    any
	Optional bool
	Suffix   string
}
