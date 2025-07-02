package cmd

// Parameter is an interface for a generic parameters. Users may have types as command parameters that
// implement this parameter.
type Parameter interface {
	// Type returns the type of the parameter. It will show up in the usage of the command, and, if one of the
	// known type names, will also show up client-side.
	Type() string
}

// Enum is an interface for enum-type parameters. Users may have types as command parameters that implement
// this parameter in order to allow a specific set of options only.
// Enum implementations must be of the type string, for example:
//
//	type GameMode string
//	func (GameMode) Type() string { return "GameMode" }
//	func (GameMode) Options(Source) []string { return []string{"survival", "creative"} }
//
// Their values will then automatically be set to whichever option returned in Enum.Options is selected by
// the user.
type Enum interface {
	// Type returns the type of the enum. This type shows up client-side in the command usage, in the spot
	// where parameter types otherwise are.
	// Type names returned are used as an identifier for this enum type. Different Enum implementations must
	// return a different string in the Type method.
	Type() string
	// Options should return a list of options that show up on the client side. The command will ensure that
	// the argument passed to the enum parameter will be equal to one of these options. The provided Source
	// can also be used to change the enums for each player.
	Options() []string
}

// SubCommand represents a subcommand that may be added as a static value that must be written. Adding
// multiple Runnable implementations to the command in New with different SubCommand fields as the
// first parameter allows for commands with subcommands.
type SubCommand struct{}

// Varargs is an argument type that may be used to capture all arguments that follow. This is useful for,
// for example, messages and names.
type Varargs string

// Optional is an argument type that may be used to make any of the available parameter types optional. Optional command
// parameters may only occur at the end of the Runnable struct. No non-optional parameter is allowed after an optional
// parameter.
type Optional[T any] struct {
	val T
	set bool
}

// Load returns the value specified upon executing the command and a bool that is true if the parameter was filled out
// by the Source.
func (o Optional[T]) Load() (T, bool) {
	return o.val, o.set
}

// LoadOr returns the value specified upon executing the command, or a value 'or' if the parameter was not filled out
// by the Source.
func (o Optional[T]) LoadOr(or T) T {
	if o.set {
		return o.val
	}
	return or
}
