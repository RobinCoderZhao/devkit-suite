package watchbot

import (
	"strings"

	"github.com/RobinCoderZhao/devkit-suite/pkg/notify"
)

// ComposeDigest creates a single aggregated notification for a subscriber.
// Uses the WatchBot formatter for channel-specific output.
func ComposeDigest(changes []Change, subscriber SubscriberWithCompetitors, formatter *notify.WatchEmailFormatter) notify.Message {
	if len(changes) == 0 {
		return notify.Message{}
	}

	// Convert internal Change to WatchBot formatter model
	items := make([]notify.WatchChangeItem, len(changes))
	for i, c := range changes {
		items[i] = notify.WatchChangeItem{
			CompetitorName: c.CompetitorName,
			PageType:       c.PageType,
			PageURL:        c.PageURL,
			Severity:       c.Severity,
			Analysis:       c.Analysis,
			Additions:      c.Additions,
			Deletions:      c.Deletions,
		}
	}

	// Find unchanged competitors
	changedCompIDs := make(map[int]bool)
	for _, c := range changes {
		for i, name := range subscriber.CompetitorNames {
			if name == c.CompetitorName && i < len(subscriber.CompetitorIDs) {
				changedCompIDs[subscriber.CompetitorIDs[i]] = true
			}
		}
	}
	var unchanged []string
	for i, name := range subscriber.CompetitorNames {
		if i < len(subscriber.CompetitorIDs) && !changedCompIDs[subscriber.CompetitorIDs[i]] {
			unchanged = append(unchanged, name)
		}
	}

	data := notify.WatchDigestData{
		ChangeCount: len(changes),
		Changes:     items,
		Unchanged:   unchanged,
		Date:        strings.Split(changes[0].DetectedAt.String(), " ")[0],
	}

	return formatter.Format(data)
}
