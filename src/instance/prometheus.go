package instance

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Prometheus interface {
	Register(prometheus.Registerer)

	CurrentJobCount() prometheus.Gauge
	TotalJobDurationSeconds() prometheus.Histogram
}
