// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelines_stats

import (
	"github.com/elastic/beats/metricbeat/helper/prometheus"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()
)

var mapping = &prometheus.MetricsMapping{
	Metrics: map[string]prometheus.MetricMap{
		"tekton_taskrun_count":              prometheus.Metric("taskrun.count"),
		"tekton_running_taskruns_count":     prometheus.Metric("taskrun.running.count"),
		"tekton_pipelinerun_count":          prometheus.Metric("pipelinerun.count"),
		"tekton_running_pipelineruns_count": prometheus.Metric("pipelinerun.running.count"),
	},

	Labels: map[string]prometheus.LabelMap{
		"status": prometheus.KeyLabel("status"),
	},
}

func init() {
	mb.Registry.MustAddMetricSet("tekton", "pipelines_stats",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(hostParser))
}
