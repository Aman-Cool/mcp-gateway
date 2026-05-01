package otel

import (
	"testing"
)

func TestNewMetricsProvider_NoEndpoint(t *testing.T) {
	cfg := &Config{
		Endpoint:       "",
		ServiceName:    "test-service",
		ServiceVersion: "v1.0.0",
	}

	_, err := NewMetricsProvider(t.Context(), cfg)
	if err == nil {
		t.Error("expected error when no endpoint configured, got nil")
	}
}

func TestNewMetricsProvider_InvalidEndpointScheme(t *testing.T) {
	cfg := &Config{
		Endpoint:       "ftp://invalid:4318",
		ServiceName:    "test-service",
		ServiceVersion: "v1.0.0",
	}

	_, err := NewMetricsProvider(t.Context(), cfg)
	if err == nil {
		t.Error("expected error for invalid scheme, got nil")
	}
}

func TestNewMetricsProvider_ValidHTTPEndpoint(t *testing.T) {
	cfg := &Config{
		Endpoint:       "http://localhost:4318",
		Insecure:       true,
		ServiceName:    "test-service",
		ServiceVersion: "v1.0.0",
	}

	provider, err := NewMetricsProvider(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected provider, got nil")
	}
	if provider.MeterProvider() == nil {
		t.Error("expected MeterProvider, got nil")
	}
	// Shutdown may fail to reach the collector in unit tests; only check for panic.
	_ = provider.Shutdown(t.Context())
}

func TestNewMetricsProvider_ValidGRPCEndpoint(t *testing.T) {
	cfg := &Config{
		Endpoint:       "rpc://localhost:4317",
		Insecure:       true,
		ServiceName:    "test-service",
		ServiceVersion: "v1.0.0",
	}

	provider, err := NewMetricsProvider(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected provider, got nil")
	}
	// Shutdown may fail to reach the collector in unit tests; only check for panic.
	_ = provider.Shutdown(t.Context())
}

func TestMetricsProvider_ShutdownNil(t *testing.T) {
	p := &MetricsProvider{meterProvider: nil}

	if err := p.Shutdown(t.Context()); err != nil {
		t.Errorf("expected nil error for nil provider, got: %v", err)
	}
}
