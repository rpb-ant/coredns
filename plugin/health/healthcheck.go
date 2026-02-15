package health

import "strings"

// HealthChecker is implemented by plugins that want to participate in the
// /health endpoint. When any registered HealthChecker returns false, the
// endpoint returns a 503 status code instead of 200.
//
// Plugins opt in purely by implementing the interface — no explicit
// registration call is needed. The health plugin discovers all handlers in
// the same server block that satisfy HealthChecker at startup.
type HealthChecker interface {
	// Healthy is called on each request to /health. It must be safe to call
	// concurrently and should return quickly.
	Healthy() bool
}

// namedChecker pairs a HealthChecker with the name of the plugin that
// implements it, used for reporting which plugins are unhealthy.
type namedChecker struct {
	name    string
	checker HealthChecker
}

// aggregateHealth returns (true, "") if all registered HealthCheckers report
// healthy, or if none are registered. Otherwise it returns (false, names)
// where names is a comma-separated list of the unhealthy plugin names.
func (h *health) aggregateHealth() (bool, string) {
	var unhealthy []string
	for _, nc := range h.checkers {
		if !nc.checker.Healthy() {
			unhealthy = append(unhealthy, nc.name)
		}
	}
	if len(unhealthy) == 0 {
		return true, ""
	}
	return false, strings.Join(unhealthy, ",")
}
