// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/logging/logadmin"
	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	trace "cloud.google.com/go/trace/apiv1"
	"cloud.google.com/go/trace/apiv1/tracepb"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	projectID = flag.String("project", "", "GCP Project ID")
	namespace = flag.String("namespace", "", "Kubernetes namespace")
	timeout   = flag.Duration("timeout", 5*time.Minute, "Timeout for the entire verification process")
	waitSec   = flag.Duration("wait", 2*time.Minute, "Time to wait for telemetry to propagate before verifying")
)

func main() {
	flag.Parse()

	if *projectID == "" {
		log.Fatal("--project is required")
	}
	if *namespace == "" {
		log.Fatal("--namespace is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	fmt.Printf("Waiting %v for telemetry to propagate...\n", *waitSec)
	select {
	case <-time.After(*waitSec):
	case <-ctx.Done():
		log.Fatalf("Timeout waiting for propagation: %v", ctx.Err())
	}

	err := runVerification(ctx)
	if err != nil {
		fmt.Printf("Verification FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Verification PASSED")
}

func runVerification(ctx context.Context) error {
	// Initialize clients
	logAdminClient, err := logadmin.NewClient(ctx, *projectID)
	if err != nil {
		return fmt.Errorf("failed to create logadmin client: %w", err)
	}
	defer logAdminClient.Close()

	traceClient, err := trace.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create trace client: %w", err)
	}
	defer traceClient.Close()

	metricClient, err := monitoring.NewMetricClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create metric client: %w", err)
	}
	defer metricClient.Close()

	// Run verifications
	fmt.Println("Verifying logs...")
	if err := retry(ctx, "logs verification", 30*time.Second, func() error {
		return verifyLogs(ctx, logAdminClient)
	}); err != nil {
		return err
	}
	fmt.Println("Logs verified successfully")

	fmt.Println("Verifying traces...")
	if err := retry(ctx, "traces verification", 30*time.Second, func() error {
		return verifyTraces(ctx, traceClient)
	}); err != nil {
		return err
	}
	fmt.Println("Traces verified successfully")

	fmt.Println("Verifying metrics...")
	if err := retry(ctx, "metrics verification", 30*time.Second, func() error {
		return verifyMetrics(ctx, metricClient)
	}); err != nil {
		return err
	}
	fmt.Println("Metrics verified successfully")

	return nil
}

func retry(ctx context.Context, desc string, interval time.Duration, fn func() error) error {
	for {
		err := fn()
		if err == nil {
			return nil
		}
		log.Printf("%s failed: %v. Retrying in %v...", desc, err, interval)
		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s: %w", desc, ctx.Err())
		}
	}
}

func verifyLogs(ctx context.Context, client *logadmin.Client) error {
	filter := fmt.Sprintf(`resource.type="k8s_container" AND resource.labels.namespace_name=%q`, *namespace)
	startTime := time.Now().Add(-10 * time.Minute).Format(time.RFC3339)
	filter += fmt.Sprintf(` AND timestamp >= %q`, startTime)

	iter := client.Entries(ctx, logadmin.Filter(filter))
	entry, err := iter.Next()
	if err == iterator.Done {
		return fmt.Errorf("no logs found matching filter: %s", filter)
	}
	if err != nil {
		return fmt.Errorf("error reading logs: %w", err)
	}
	fmt.Printf("Found log entry: %v\n", entry.Payload)
	return nil
}

func verifyTraces(ctx context.Context, client *trace.Client) error {
	log.Println("DEBUG: Searching for namespace in all traces...")
	req := &tracepb.ListTracesRequest{
		ProjectId: *projectID,
		StartTime: timestamppb.New(time.Now().Add(-10 * time.Minute)),
		EndTime:   timestamppb.New(time.Now()),
		View:      tracepb.ListTracesRequest_COMPLETE,
	}
	iter := client.ListTraces(ctx, req)
	for {
		tr, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("error listing traces: %w", err)
		}
		for _, span := range tr.Spans {
			for k, v := range span.Labels {
				if v == *namespace {
					log.Printf("DEBUG: Found namespace %s in label %s of span %s (trace %s)", *namespace, k, span.Name, tr.TraceId)
				}
				if k == "k8s.namespace.name" && v == *namespace {
					fmt.Printf("Found trace: %s (via k8s.namespace.name check in Go)\n", tr.TraceId)
					return nil
				}
			}
		}
	}

	return fmt.Errorf("no traces found for namespace %s (debug run)", *namespace)
}

func verifyMetrics(ctx context.Context, client *monitoring.MetricClient) error {
	metricTypes := []string{
		"prometheus.googleapis.com/demo.recommendation.requests/counter",
		"prometheus.googleapis.com/demo.ad.requests/counter",
		"prometheus.googleapis.com/http.server.request.duration/histogram",
	}
	fmt.Println("Found TimeSeries:")
	count := 0
	matchedCount := 0
	var lastErr error
	for _, metricType := range metricTypes {
		req := &monitoringpb.ListTimeSeriesRequest{
			Name:   fmt.Sprintf("projects/%s", *projectID),
			Filter: fmt.Sprintf(`metric.type = %q`, metricType),
			Interval: &monitoringpb.TimeInterval{
				StartTime: timestamppb.New(time.Now().Add(-10 * time.Minute)),
				EndTime:   timestamppb.New(time.Now()),
			},
		}
		iter := client.ListTimeSeries(ctx, req)
		for {
			ts, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				lastErr = fmt.Errorf("error listing metrics for %s: %w", metricType, err)
				break
			}
			ns := ts.Resource.Labels["namespace"]
			job := ts.Resource.Labels["job"]

			isMatched := ns == *namespace ||
				(ns == "" && (fmt.Sprintf("%s", ts.Resource.Labels) == *namespace ||
					fmt.Sprintf("%s", ts.Metric.Labels) == *namespace)) ||
				(job != "" && (job == *namespace || strings.HasPrefix(job, *namespace+"/")))

			if isMatched {
				fmt.Printf("MATCHED (%s) - Job: %s, ResourceLabels: %v, ResourceType: %s, MetricLabels: %v\n",
					metricType, job, ts.Resource.Labels, ts.Resource.Type, ts.Metric.Labels)
				matchedCount++
			}
			count++
		}
		if matchedCount > 0 {
			break
		}
	}
	fmt.Printf("Total timeseries checked: %d, Matched: %d\n", count, matchedCount)
	if matchedCount > 0 {
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("no metrics found for namespace %s (last error: %w)", *namespace, lastErr)
	}
	return fmt.Errorf("no metrics found for namespace %s", *namespace)
}
