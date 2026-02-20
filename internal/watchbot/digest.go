package watchbot

import (
	"strings"

	"github.com/RobinCoderZhao/devkit-suite/pkg/notify"
)

// ComposeDigest creates a single aggregated notification for a subscriber.
// Changes are grouped by competitor for better readability.
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

	// Group changes by competitor
	groups := notify.GroupChanges(items)

	// Find unchanged competitors
	changedCompNames := make(map[string]bool)
	for _, c := range changes {
		changedCompNames[c.CompetitorName] = true
	}
	var unchanged []string
	seen := make(map[string]bool)
	for _, name := range subscriber.CompetitorNames {
		if !changedCompNames[name] && !seen[name] {
			unchanged = append(unchanged, name)
			seen[name] = true
		}
	}

	data := notify.WatchDigestData{
		ChangeCount: len(changes),
		Groups:      groups,
		Unchanged:   unchanged,
		Date:        strings.Split(changes[0].DetectedAt.String(), " ")[0],
	}

	return formatter.Format(data)
}
