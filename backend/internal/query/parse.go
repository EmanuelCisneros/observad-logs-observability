package query

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/observa/observad/internal/model"
)

type Params struct {
	From     string `json:"from" form:"from"`
	To       string `json:"to" form:"to"`
	Levels   string `json:"levels" form:"levels"`
	Services string `json:"services" form:"services"`
	Search   string `json:"search" form:"search"`
	Limit    string `json:"limit" form:"limit"`
	Offset   string `json:"offset" form:"offset"`
	Order    string `json:"order" form:"order"`
}

func Parse(p Params, now time.Time) (model.LogQuery, error) {
	var q model.LogQuery

	from, err := parseTime(p.From, now)
	if err != nil {
		return q, fmt.Errorf("invalid 'from': %w", err)
	}
	q.From = from

	to, err := parseTime(p.To, now)
	if err != nil {
		return q, fmt.Errorf("invalid 'to': %w", err)
	}
	q.To = to

	if !q.From.IsZero() && !q.To.IsZero() && q.From.After(q.To) {
		return q, fmt.Errorf("'from' must not be after 'to'")
	}

	q.Levels, err = parseLevels(p.Levels)
	if err != nil {
		return q, err
	}
	q.Services = splitCSV(p.Services)
	q.Search = strings.TrimSpace(p.Search)

	if p.Limit != "" {
		n, err := strconv.Atoi(p.Limit)
		if err != nil {
			return q, fmt.Errorf("invalid 'limit': must be a number")
		}
		q.Limit = n
	}
	if p.Offset != "" {
		n, err := strconv.Atoi(p.Offset)
		if err != nil {
			return q, fmt.Errorf("invalid 'offset': must be a number")
		}
		q.Offset = n
	}
	q.Order = strings.ToLower(strings.TrimSpace(p.Order))

	q.Clamp()
	return q, nil
}

// parseTime: empty, RFC3339, or duration relative to now.
func parseTime(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		if d < 0 {
			d = -d
		}
		return now.Add(-d), nil
	}
	return time.Time{}, fmt.Errorf("expected RFC3339 timestamp or duration like '15m'")
}

func parseLevels(s string) ([]model.Level, error) {
	parts := splitCSV(s)
	if len(parts) == 0 {
		return nil, nil
	}
	levels := make([]model.Level, 0, len(parts))
	for _, p := range parts {
		lvl := model.Level(strings.ToLower(p))
		if !lvl.Valid() {
			return nil, fmt.Errorf("unknown level %q", p)
		}
		levels = append(levels, lvl)
	}
	return levels, nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		if v := strings.TrimSpace(r); v != "" {
			out = append(out, v)
		}
	}
	return out
}
