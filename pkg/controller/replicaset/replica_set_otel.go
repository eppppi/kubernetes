// MADE BY: @eppppi
// REF: https://blog.cybozu.io/entry/2023/04/12/170000

package replicaset

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	// "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

// define tracer as global variable
var tracer = otel.Tracer("k8s.io/kubernetes/pkg/controller/replicaset")

const serviceName = "k8s.io/kubernetes/pkg/controller/replicaset"

func DoSomething(ctx context.Context, foo string, bar string) error {
	_, span := tracer.Start(ctx, "Here is replicaset", trace.WithAttributes(
		attribute.String("foo", foo),
		attribute.String("bar", bar),
	))
	time.Sleep(time.Millisecond * 10)
	defer span.End()

	return nil
}

func setupTracer() {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		panic(err)
	}
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		attribute.Key("service.namespace").String(serviceName),
	)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
}

// endpoint := "localhost:4317"
// exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure(), otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithDialOption(grpc.WithBlock()))
