package main

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

const serviceName = "koc-trace-cm"

func setupTracer() {
	exporter, err := setupOtlpExporter()
	if err != nil {
		panic(err)
	}
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		attribute.Key("service.name").String(serviceName),
	)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tp)
}

func setupOtlpExporter() (sdktrace.SpanExporter, error) {
	endpoint := "jaeger.jaeger.svc.cluster.local:4318"
	return otlptracehttp.New(context.Background(), otlptracehttp.WithInsecure(), otlptracehttp.WithEndpoint(endpoint))
}

func writeResolvconf() {
	resolvconf := `search default.svc.cluster.local svc.cluster.local cluster.local
nameserver 10.96.0.10
options ndots:5
`
	fp, err := os.Create("/etc/resolv.conf")
	if err != nil {
		fmt.Println("failed to open resolv.conf: ", err)
		return
	}
	defer fp.Close()
	_, err = fp.Write([]byte(resolvconf))
	if err != nil {
		fmt.Println("failed to write resolv.conf: ", err)
		return
	}
	fmt.Println("success to write resolv.conf")
}
