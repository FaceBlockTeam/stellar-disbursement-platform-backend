package monitor

import "github.com/prometheus/client_golang/prometheus"

var SummaryVecMetrics = map[MetricTag]*prometheus.SummaryVec{
	HttpRequestDurationTag: prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "sdp", Subsystem: "http", Name: string(HttpRequestDurationTag),
		Help: "HTTP requests durations, sliding window = 10m",
	},
		[]string{"status", "route", "method"},
	),
	SuccessfulQueryDurationTag: prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "sdp", Subsystem: "db", Name: string(SuccessfulQueryDurationTag),
		Help: "Successful DB query durations",
	},
		[]string{"query_type"},
	),
	FailureQueryDurationTag: prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "sdp", Subsystem: "db", Name: string(FailureQueryDurationTag),
		Help: "Failure DB query durations",
	},
		[]string{"query_type"},
	),
}

var CounterMetrics map[MetricTag]prometheus.Counter

var HistogramVecMetrics map[MetricTag]prometheus.HistogramVec

var CounterVecMetrics = map[MetricTag]*prometheus.CounterVec{
	DisbursementsCounterTag: prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "sdp", Subsystem: "bussiness", Name: string(DisbursementsCounterTag),
		Help: "Disbursements Counter",
	},
		[]string{"asset", "country", "wallet"},
	),
}
