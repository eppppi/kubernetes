// MADE BY: @eppppi

package apply

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	// "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

// define tracer as global variable
var tracer = otel.Tracer("k8s.io/kubernetes/pkg/controller/kubectl-apply")

const serviceName = "k8s.io/kubernetes/pkg/controller/kubectl-apply"

func DoSomething(ctx context.Context, foo string, bar string) error {
	fmt.Println("DoSomething in kubectl-apply")
	_, span := tracer.Start(ctx, "Here is kubectl", trace.WithAttributes(
		attribute.String("foo-kubectl-apply", foo),
		attribute.String("bar-kubectl-apply", bar),
	))
	defer span.End()

	return nil
}

func setupTracer() func() {
	exporter, err := setupOtlpExporter()
	if err != nil {
		panic(err)
	}
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		// attribute.Key("service.namespace").String(serviceName),
		attribute.Key("service.name").String(serviceName),
	)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := tp.ForceFlush(ctx); err != nil {
			fmt.Println(err)
		}
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		if err := tp.Shutdown(ctx2); err != nil {
			fmt.Println(err)
		}
		cancel()
		cancel2()
	}
	fmt.Println("setting up tracer provider")
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tp)
	return cleanup
}

func setupOtlpExporter() (sdktrace.SpanExporter, error) {
	endpoint := "localhost:4318"
	return otlptracehttp.New(context.Background(), otlptracehttp.WithInsecure(), otlptracehttp.WithEndpoint(endpoint))
}

func debugPostTrace() {
	jsonBody := `{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"test-with-curl-in-kubectl-apply"}}]},"scopeSpans":[{"scope":{"name":"manual-test"},"spans":[{"traceId":"71699b6fe85982c7c8995ea3d9c95df2","spanId":"3c191d03fa8be065","name":"spanitron","kind":2,"droppedAttributesCount":0,"events":[],"droppedEventsCount":0,"status":{"code":1}}]}]}]}`

	fmt.Println("posting to jaeger")
	// resp, err := http.Post("http://jaeger.jaeger.svc.cluster.local:4318/v1/traces", "application/json", bytes.NewBuffer([]byte(jsonBody)))
	resp, err := http.Post("http://localhost:4318/v1/traces", "application/json", bytes.NewBuffer([]byte(jsonBody)))
	if err != nil {
		fmt.Println("post failed: ", err)
		return
	} else {
		fmt.Println("post success: ", resp)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println(string(body))
}
