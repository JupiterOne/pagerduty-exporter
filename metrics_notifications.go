package main

import (
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/prometheus/client_golang/prometheus"
	prometheusCommon "github.com/webdevops/go-common/prometheus"
	"github.com/webdevops/go-common/prometheus/collector"
)

type MetricsCollectorNotifications struct {
	collector.Processor

	prometheus struct {
		notification *prometheus.GaugeVec
	}
}

func (m *MetricsCollectorNotifications) Setup(collector *collector.Collector) {
	m.Processor.Setup(collector)

	m.prometheus.notification = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pagerduty_notification_info",
			Help: "PagerDuty Notification",
		},
		[]string{
			"ID",
			"address",
			"status",
			"type",
			"userID",
			"userSummary",
			"time",
		},
	)

	prometheus.MustRegister(m.prometheus.notification)
}

func (m *MetricsCollectorNotifications) Reset() {
	m.prometheus.notification.Reset()
}

func (m *MetricsCollectorNotifications) Collect(callback chan<- func()) {
	now := time.Now().UTC()

	listOpts := pagerduty.ListNotificationOptions{
		Limit:  PagerdutyListLimit,
		Offset: 0,
		Since:  now.Add(-opts.PagerDuty.Notification.Since).Format(time.RFC3339),
		Until:  now.Format(time.RFC3339),
	}

	notificationMetricList := prometheusCommon.NewMetricsList()

	for {
		m.Logger().Debugf("fetch notifications (offset:%v, limit:%v)", listOpts.Offset, listOpts.Limit)

		list, err := PagerDutyClient.ListNotificationsWithContext(m.Context(), listOpts)
		PrometheusPagerDutyApiCounter.WithLabelValues("ListNotifications").Inc()

		if err != nil {
			m.Logger().Panic(err)
		}

		for _, notification := range list.Notifications {
			createdAt, _ := time.Parse(time.RFC3339, notification.StartedAt)

			notificationMetricList.AddTime(prometheus.Labels{
				"ID":          notification.ID,
				"address":     notification.Address,
				"status":      notification.Status,
				"type":        notification.Type,
				"userID":      notification.User.ID,
				"userSummary": notification.User.Summary,
				"time":        createdAt.Format(opts.PagerDuty.Notification.TimeFormat),
			}, createdAt)
		}

		listOpts.Offset += PagerdutyListLimit
		if !list.More || listOpts.Offset >= opts.PagerDuty.Notification.Limit {
			break
		}
	}

	callback <- func() {
		notificationMetricList.GaugeSet(m.prometheus.notification)
	}
}
