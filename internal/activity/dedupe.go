package activity

import "time"

func Deduplicate(events []Event) []Event {
	if len(events) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(events))
	result := make([]Event, 0, len(events))
	for _, event := range events {
		key := event.Key()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, event)
	}
	Sort(result)
	return result
}

func ExcludeTimestamps(events []Event, timestamps []time.Time) []Event {
	if len(events) == 0 {
		return nil
	}
	if len(timestamps) == 0 {
		copied := append([]Event(nil), events...)
		Sort(copied)
		return copied
	}

	excluded := make(map[string]struct{}, len(timestamps))
	for _, timestamp := range timestamps {
		excluded[normalizeTimestamp(timestamp)] = struct{}{}
	}

	result := make([]Event, 0, len(events))
	for _, event := range events {
		if _, ok := excluded[normalizeTimestamp(event.Timestamp)]; ok {
			continue
		}
		result = append(result, event)
	}
	Sort(result)
	return result
}
