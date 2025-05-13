package session

// Context ...
type Context struct {
	cancel bool
}

// NewContext ...
func NewContext() *Context {
	return &Context{}
}

// Cancelled ...
func (ctx *Context) Cancelled() bool {
	return ctx.cancel
}

// Cancel ...
func (ctx *Context) Cancel() {
	ctx.cancel = true
}
