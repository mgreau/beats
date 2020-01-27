// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package taskruns

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
		"tekton_taskrun_count":            prometheus.Metric("total"),
		"tekton_running_taskruns_count":   prometheus.Metric("running"),
		"tekton_taskrun_duration_seconds": prometheus.Metric("taskrun.duration.sec", prometheus.OpMultiplyBuckets(1000)),
	},

	Labels: map[string]prometheus.LabelMap{
		"task":      prometheus.KeyLabel("task"),
		"taskrun":   prometheus.KeyLabel("taskrun"),
		"namespace": prometheus.KeyLabel("namespace"),
	},
}

func init() {
	mb.Registry.MustAddMetricSet("tekton", "taskruns",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(hostParser))
}
