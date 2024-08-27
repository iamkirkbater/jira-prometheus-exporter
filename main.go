package main

import (
	"flag"
	"fmt"
	log "log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/iamkirkbater/jira-exporter/pkg/jira"
)

const (
	// Time in seconds to sleep between queries
	TIME_BETWEEN_JIRA_QUERIES = 60

	// Base URL of the JIRA instance to query from
	JIRA_BASE_URL = "https://issues.redhat.com"
)

var (
	logLevel   string
	jiraClient jira.Client
	issueGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ohss_issues",
		Help: "The total number of OHSS issues on the board, including recently resolved",
	},
		[]string{
			"priority",
			"status",
		},
	)
	breachingSLAGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ohss_issues_breaching_sla",
		Help: "The total number of OHSS issues breaching SLA, labeled by priority and status",
	},
		[]string{
			"priority",
		},
	)
)

func getMetrics(c jira.Client) {
	for {
		calculateMetrics(c)

		log.Debug(fmt.Sprintf("Sleeping for %d seconds...", TIME_BETWEEN_JIRA_QUERIES))
		time.Sleep(TIME_BETWEEN_JIRA_QUERIES * time.Second)
	}
}

func calculateMetrics(c jira.Client) error {
	log.Debug("Querying JIRA for OHSS issues")
	issues, err := c.GetIssues()
	if err != nil {
		log.Error("There was an error querying JIRA")
		return err
	}

	// Since we're using `.Inc` in the for loop below to only increment by one every time, we'll specifically
	// want to use Reset here in order to make sure the count for all gauges is 0 before we start incrementing
	// otherwise we'll inherit whatever the count was from the last run of the func and increment that instead.
	issueGauge.Reset()
	breachingSLAGauge.Reset()

	for _, issue := range issues {
		issueGauge.With(prometheus.Labels{"priority": issue.Priority, "status": issue.Status}).Inc()

		if isBreachingSLA(issue) {
			breachingSLAGauge.With(prometheus.Labels{"priority": issue.Priority}).Inc()
		}
	}
	log.Debug("Gauges Updated")
	return nil
}

// SLA is defined as the following:
// 4 Hour Update Window for SRE to respond to Urgent and High tickets
// 24 Hour update window for SRE to respond to Medium tickets
// 72 Hour update window for SRE to respond to Low tickets
func isBreachingSLA(issue *jira.Issue) bool {
	// SLA is not valid if not awaiting SRE response, so anything that
	// is pending customer, pending vendor, etc is not "SLA'd" here.
	// So that really only leaves us In Progress and New tickets, so we
	// exit early for other statuses:
	if issue.Status != "in_progress" && issue.Status != "new" {
		return false
	}

	timeForSLABreach := time.Now()

	switch issue.Priority {
	case "urgent", "high":
		// Idk why you can't Time.Sub() a duration but here we are
		timeForSLABreach.Add(-4 * time.Hour)

	case "medium":
		timeForSLABreach.Add(-24 * time.Hour)

	case "low":
		timeForSLABreach.Add(-72 * time.Hour)
	}

	return issue.LastUpdatedTime.Before(timeForSLABreach)
}

// This function handles registering all Prometheus metrics for OHSS issue tracking
func registerMetrics() {
	prometheus.MustRegister(issueGauge)
	prometheus.MustRegister(breachingSLAGauge)
}

func main() {
	jiraClient, err := jira.NewClient(JIRA_BASE_URL)
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

	registerMetrics()

	// We want to make sure the initial calculation happens before we serve metrics
	// in the case that the prom exporter is still coming online when it gets scraped
	// so the metrics don't disappear. This leads to a bit longer of a startup time
	// but should hopefully mean more consistent metrics
	calculateMetrics(jiraClient)

	// start the cyclical metrics gatherer
	go func() {
		// since we _just_ called the calculate metrics on initializaiton sleep for the
		// same duration to not double up on the JIRA queries
		time.Sleep(TIME_BETWEEN_JIRA_QUERIES * time.Second)
		getMetrics(jiraClient)
	}()

	// Register the HTTP Handler the metrics now that the initial metrics have been gathered
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8000", nil)
}

func init() {
	// Initialze Logger
	flag.StringVar(&logLevel, "log-level", "info", "Log Level to display. Default is info, availble vars are debug, info, warning, error")

	flag.Parse()

	logLevelMap := map[string]log.Level{
		"debug":   log.LevelDebug,
		"info":    log.LevelInfo,
		"warn":    log.LevelWarn,
		"warning": log.LevelWarn,
		"err":     log.LevelError,
		"error":   log.LevelError,
	}

	if level, ok := logLevelMap[logLevel]; !ok {
		log.Error(fmt.Sprintf("non-valid log level '%s' entered. Available values are debug, info, warning, error", level))
		os.Exit(1)
	}

	opts := &log.HandlerOptions{
		Level: logLevelMap[logLevel],
	}

	handler := log.NewTextHandler(os.Stdout, opts)
	logger := log.New(handler)
	log.SetDefault(logger)
}
