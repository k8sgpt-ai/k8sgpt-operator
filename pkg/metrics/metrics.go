/*
Copyright 2023 The K8sGPT Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// MetricType represents the type of Prometheus metric (Counter or Gauge)
type MetricType int

const (
	Counter MetricType = iota
	Gauge
)

// MetricConfig holds the configuration for a single metric
type MetricConfig struct {
	Name   string
	Help   string
	Labels []string
	Type   MetricType
}

// MetricBuilder helps in building and registering Prometheus metrics
type MetricBuilder struct {
	metrics map[string]interface{}
}

// NewMetricBuilder creates a new MetricBuilder instance
func NewMetricBuilder() *MetricBuilder {
	return &MetricBuilder{
		metrics: make(map[string]interface{}),
	}
}

// AddMetric adds a metric to the builder
func (b *MetricBuilder) AddMetric(config MetricConfig) *MetricBuilder {
	switch config.Type {
	case Counter:
		b.metrics[config.Name] = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: config.Name,
			Help: config.Help,
		}, config.Labels)
	case Gauge:
		b.metrics[config.Name] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: config.Name,
			Help: config.Help,
		}, config.Labels)
	}
	return b
}

// RegisterMetrics registers all metrics with Prometheus
func (b *MetricBuilder) RegisterMetrics() {
	for _, metric := range b.metrics {
		switch m := metric.(type) {
		case *prometheus.CounterVec:
			prometheus.MustRegister(m)
		case *prometheus.GaugeVec:
			prometheus.MustRegister(m)
		}
	}
}

// GetCounterVec returns a specific CounterVec metric by name
func (b *MetricBuilder) GetCounterVec(name string) *prometheus.CounterVec {
	if metric, ok := b.metrics[name].(*prometheus.CounterVec); ok {
		return metric
	}
	return nil
}

// GetGaugeVec returns a specific GaugeVec metric by name
func (b *MetricBuilder) GetGaugeVec(name string) *prometheus.GaugeVec {
	if metric, ok := b.metrics[name].(*prometheus.GaugeVec); ok {
		return metric
	}
	return nil
}

// Usage example
func InitializeMetrics() *MetricBuilder {
	builder := NewMetricBuilder()

	builder.AddMetric(MetricConfig{
		Name:   "k8sgpt_reconcile_error_count",
		Help:   "The total number of errors during reconcile",
		Labels: []string{"k8sgpt"},
		Type:   Counter,
	}).AddMetric(MetricConfig{
		Name:   "k8sgpt_number_of_results",
		Help:   "The total number of results",
		Labels: []string{"k8sgpt"},
		Type:   Gauge,
	}).AddMetric(MetricConfig{
		Name:   "k8sgpt_number_of_results_by_type",
		Help:   "The total number of results by type",
		Labels: []string{"kind", "name", "k8sgpt"},
		Type:   Gauge,
	}).AddMetric(MetricConfig{
		Name:   "k8sgpt_number_of_backend_ai_calls",
		Help:   "The total number of backend AI calls",
		Labels: []string{"backend", "deployment", "namespace", "k8sgpt"},
		Type:   Counter,
	}).AddMetric(MetricConfig{
		Name:   "k8sgpt_number_of_failed_backend_ai_calls",
		Help:   "The total number of failed backend AI calls",
		Labels: []string{"backend", "deployment", "namespace", "k8sgpt"},
		Type:   Counter,
	}).AddMetric(MetricConfig{
		Name:   "k8sgpt_mutations_count",
		Help:   "The total number of mutations",
		Labels: []string{"mutations", "k8sgpt"},
		Type:   Gauge,
	})

	builder.RegisterMetrics()

	return builder
}
