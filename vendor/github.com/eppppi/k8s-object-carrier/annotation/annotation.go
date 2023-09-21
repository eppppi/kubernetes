package annotation

import (
	"fmt"

	"github.com/google/uuid"
)

// Here contains the ids required for otel propagation
type ObjContext struct {
	traceId  string
	parentId string
	spanId   string
}

type ObjContexts []ObjContext

// String is a method to print the context
func (c *ObjContext) String() string {
	return fmt.Sprintf("traceId: %s, parentId: %s, spanId: %s", c.traceId, c.parentId, c.spanId)
}

// CreateChildContext is a method to create a child context
func (c *ObjContext) CreateChildContext() ObjContext {
	return ObjContext{
		traceId:  c.traceId,
		parentId: c.spanId,
		spanId:   generateSpanId(),
	}
}

// String is a method to print the contexts
func (cs *ObjContexts) String() string {
	s := ""
	for _, c := range *cs {
		s += c.String() + "\n"
	}
	return s
}

// GetTraceId is a method to get the traceId
func (c *ObjContext) GetTraceId() string {
	return c.traceId
}

// GetParentId is a method to get the parentId
func (c *ObjContext) GetParentId() string {
	return c.parentId
}

// GetSpanId is a method to get the spanId
func (c *ObjContext) GetSpanId() string {
	return c.spanId
}

// TODO: replace to the off-the-shelf library in OpenTelemetry
// generateTraceId is a method to generate a trace id
func generateTraceId() string {
	id := uuid.New()
	return id.String()[:16]
}

// TODO: replace to the off-the-shelf library in OpenTelemetry
// generateSpanId is a method to generate a span id
func generateSpanId() string {
	id := uuid.New()
	return id.String()[:16]
}

// NewRootContext is a method to create a root context
func NewRootContext() ObjContext {
	return ObjContext{
		traceId:  generateTraceId(),
		parentId: "",
		spanId:   generateSpanId(),
	}
}
