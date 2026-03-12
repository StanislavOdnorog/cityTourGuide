package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// StoriesPlayedTotal counts the total number of story listening events.
	StoriesPlayedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "stories_played_total",
		Help: "Total number of story play/listen events.",
	})

	// CitiesDownloadedTotal counts the total number of city download manifest requests.
	CitiesDownloadedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cities_downloaded_total",
		Help: "Total number of city download manifest requests.",
	})

	// AccountsCreatedTotal counts the total number of new account registrations.
	AccountsCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "accounts_created_total",
		Help: "Total number of accounts created.",
	})

	// PushNotificationsSentTotal counts the total number of push notifications sent.
	PushNotificationsSentTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "push_notifications_sent_total",
		Help: "Total number of push notifications successfully sent.",
	})
)
