package ctx

import (
	"context"

	"github.com/google/uuid"
)

func (ctx Context) WithNewWorkflow() Context {
	return ctx.WithWorkflow(Workflow(uuid.NewString()))
}

func (ctx Context) WithWorkflow(workflow Workflow) Context {
	return Context{
		context.WithValue(
			ctx.Context,
			contextKeyWorkflow,
			workflow,
		),
	}.WithLogger(
		ctx.Logger().With(loggerKeyWorkflow, workflow),
	)
}

func (ctx Context) Workflow() Workflow {
	obj := ctx.Value(contextKeyWorkflow)
	if obj == nil {
		return ""
	}
	return obj.(Workflow)
}
