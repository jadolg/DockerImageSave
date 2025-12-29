package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	errorsTotalMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dockerimagesave_errors_total",
		Help: "The total number of errors found",
	})
	pullsCountMetric = promauto.NewCounter(prometheus.CounterOpts{
		Name: "dockerimagesave_pulls_total",
		Help: "The total number of docker pulls",
	})
)
