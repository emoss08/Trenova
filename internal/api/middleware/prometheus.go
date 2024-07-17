// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// Define the Prometheus metrics
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	requestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "Size of HTTP requests in bytes.",
			Buckets: prometheus.ExponentialBuckets(100, 10, 5),
		},
		[]string{"method", "path"},
	)
	responseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "Size of HTTP responses in bytes.",
			Buckets: prometheus.ExponentialBuckets(100, 10, 5),
		},
		[]string{"method", "path"},
	)
	httpRequestsInProgress = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_progress",
			Help: "Number of HTTP requests in progress.",
		},
		[]string{"method", "path"},
	)
	httpResponseStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_response_status_total",
			Help: "Total number of HTTP response status codes.",
		},
		[]string{"status"},
	)
)

func init() {
	// Register the metrics with Prometheus only once
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(requestSize)
	prometheus.MustRegister(responseSize)
	prometheus.MustRegister(httpRequestsInProgress)
	prometheus.MustRegister(httpResponseStatus)
}

// PrometheusMiddleware is a Fiber middleware that collects Prometheus metrics
func PrometheusMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		method := c.Method()
		path := c.Path()
		httpRequestsInProgress.WithLabelValues(method, path).Inc()
		defer httpRequestsInProgress.WithLabelValues(method, path).Dec()

		// Process the request
		err := c.Next()

		// Update the metrics
		duration := time.Since(start).Seconds()
		httpRequestsTotal.WithLabelValues(method, path).Inc()
		httpRequestDuration.WithLabelValues(method, path).Observe(duration)
		requestSize.WithLabelValues(method, path).Observe(float64(len(c.Request().Body())))
		responseSize.WithLabelValues(method, path).Observe(float64(len(c.Response().Body())))
		httpResponseStatus.WithLabelValues(strconv.Itoa(c.Response().StatusCode())).Inc()

		return err
	}
}
