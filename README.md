# Prometheus JIRA Exporter

This is a prometheus exporter built to get specific issues to track the severity, priority, and whether they're "breaching our SLA" for updates.

Get a JIRA token and set the env var `JIRA_API_TOKEN` and then run with `go run main.go`.

Available flags are `--log-level` which allows you to change the log level. The default level is info, accepted values are "debug", "info", "warning", and "error". 

There are very few logs presented by the info level by design. Due to the simplicity of the exporter the info logs will most likely just be noise and take up log storage space.

Roadmap:
[ ] - add metrics to the JIRA query itself
[ ] - allow JIRA token to be passed in as a file (mounted k8s secret support)
[ ] - add unit tests where appropriate
