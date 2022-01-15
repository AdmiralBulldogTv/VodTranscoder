package prometheus

import (
	"github.com/AdmiralBulldogTv/VodTranscoder/src/configure"
	"github.com/AdmiralBulldogTv/VodTranscoder/src/instance"

	"github.com/prometheus/client_golang/prometheus"
)

type mon struct {
	currentJobs             prometheus.Gauge
	totalJobDurationSeconds prometheus.Histogram
}

func (m *mon) Register(r prometheus.Registerer) {
	r.MustRegister(
		m.currentJobs,
		m.totalJobDurationSeconds,
	)
}

func (m *mon) CurrentJobCount() prometheus.Gauge {
	return m.currentJobs
}

func (m *mon) TotalJobDurationSeconds() prometheus.Histogram {
	return m.totalJobDurationSeconds
}

func LabelsFromKeyValue(kv []configure.KeyValue) prometheus.Labels {
	mp := prometheus.Labels{}

	for _, v := range kv {
		mp[v.Key] = v.Value
	}

	return mp
}

func New(opts SetupOptions) instance.Prometheus {
	return &mon{
		currentJobs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "transcoder_current_jobs",
			Help:        "The number of jobs being processed",
			ConstLabels: opts.Labels,
		}),
		totalJobDurationSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:        "transcoder_total_job_duration_seconds",
			Help:        "The total seconds occupied processing jobs",
			ConstLabels: opts.Labels,
		}),
	}
}

type SetupOptions struct {
	Labels prometheus.Labels
}
