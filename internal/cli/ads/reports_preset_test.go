package ads

import (
	"strings"
	"testing"
	"time"
)

func TestReportPresetDateRangeLastDays(t *testing.T) {
	start, end, err := reportPresetDateRange("", "", 7, time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC), time.UTC)
	if err != nil {
		t.Fatalf("reportPresetDateRange() error: %v", err)
	}
	if start != "2026-05-25" || end != "2026-05-31" {
		t.Fatalf("range = %s..%s, want 2026-05-25..2026-05-31", start, end)
	}
}

func TestReportsPresetCommandHelpShowsOperatorGuidance(t *testing.T) {
	cmd := ReportsPresetCommand()
	if !strings.Contains(cmd.ShortHelp, "Build and run Apple Ads report presets without JSON payloads.") {
		t.Fatalf("ShortHelp = %q, want preset workflow wording", cmd.ShortHelp)
	}
	for _, want := range []string{
		"Choose the report resource with --level.",
		"today\" is calculated in\n--time-zone",
		"Ad-level reports require --sort",
		"asc ads reports preset --level ads --campaign 12345 --from 2026-05-01 --to 2026-05-31 --sort impressions:desc",
	} {
		if !strings.Contains(cmd.LongHelp, want) {
			t.Fatalf("LongHelp missing %q\n%s", want, cmd.LongHelp)
		}
	}
	if got := cmd.FlagSet.Lookup("last-days").Usage; got != "Use an inclusive range ending today in --time-zone" {
		t.Fatalf("--last-days usage = %q", got)
	}
	if got := cmd.FlagSet.Lookup("time-zone").Usage; got != "IANA reporting time zone" {
		t.Fatalf("--time-zone usage = %q", got)
	}
}

func TestBuildReportPresetPayloadLastDaysUsesReportingTimeZone(t *testing.T) {
	payload, err := buildReportPresetPayload(reportPresetTestFlags(
		"campaigns",
		"",
		"",
		"",
		"",
		1,
		"DAILY",
		"",
		"",
		1000,
		0,
		"America/Los_Angeles",
		false,
	), time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("buildReportPresetPayload() error: %v", err)
	}
	if payload.StartTime != "2026-05-31" || payload.EndTime != "2026-05-31" {
		t.Fatalf("range = %s..%s, want 2026-05-31..2026-05-31", payload.StartTime, payload.EndTime)
	}
	if payload.TimeZone != "America/Los_Angeles" {
		t.Fatalf("timeZone = %q, want America/Los_Angeles", payload.TimeZone)
	}
}

func TestBuildReportPresetPayloadValidatesTimeZone(t *testing.T) {
	_, err := buildReportPresetPayload(reportPresetTestFlags(
		"campaigns",
		"",
		"",
		"",
		"",
		1,
		"DAILY",
		"",
		"",
		1000,
		0,
		"Pacific/Atlantis",
		false,
	), time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC))
	if err == nil || !strings.Contains(err.Error(), "--time-zone must be a valid IANA time zone") {
		t.Fatalf("error = %v, want time-zone validation", err)
	}
}

func TestBuildReportPresetPayloadRequiresSortForAds(t *testing.T) {
	_, err := buildReportPresetPayload(reportPresetTestFlags(
		"ads",
		"12345",
		"",
		"2026-05-01",
		"2026-05-31",
		0,
		"DAILY",
		"",
		"",
		1000,
		0,
		"UTC",
		false,
	), time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC))
	if err == nil || !strings.Contains(err.Error(), "--sort is required for --level ads") {
		t.Fatalf("error = %v, want ad-level sort validation", err)
	}
}

func TestReportPresetDateRangeValidation(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		days    int
		wantErr string
	}{
		{name: "last days conflicts with explicit range", from: "2026-05-01", days: 7, wantErr: "--last-days cannot be combined"},
		{name: "negative last days", days: -1, wantErr: "--last-days must be >= 0"},
		{name: "missing range", wantErr: "either --last-days or both --from and --to are required"},
		{name: "bad from", from: "2026/05/01", to: "2026-05-31", wantErr: "--from must be in YYYY-MM-DD format"},
		{name: "reversed range", from: "2026-06-01", to: "2026-05-31", wantErr: "--to must be on or after --from"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := reportPresetDateRange(tt.from, tt.to, tt.days, time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC), time.UTC)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want contains %q", err, tt.wantErr)
			}
		})
	}
}

func reportPresetTestFlags(level, campaign, adGroup, from, to string, lastDays int, granularity, fields, sort string, limit, offset int, timeZone string, returnRowTotals bool) adsReportPresetFlags {
	return adsReportPresetFlags{
		level:           &level,
		campaign:        &campaign,
		adGroup:         &adGroup,
		from:            &from,
		to:              &to,
		lastDays:        &lastDays,
		granularity:     &granularity,
		fields:          &fields,
		sort:            &sort,
		limit:           &limit,
		offset:          &offset,
		timeZone:        &timeZone,
		returnRowTotals: &returnRowTotals,
	}
}

func TestParseReportPresetSort(t *testing.T) {
	sortSpec, err := parseReportPresetSort("impressions:asc")
	if err != nil {
		t.Fatalf("parseReportPresetSort() error: %v", err)
	}
	if sortSpec.Field != "impressions" || sortSpec.SortOrder != "ASCENDING" {
		t.Fatalf("sort = %+v, want impressions ASCENDING", sortSpec)
	}

	_, err = parseReportPresetSort("impressions:sideways")
	if err == nil || !strings.Contains(err.Error(), "--sort direction must be asc or desc") {
		t.Fatalf("error = %v, want sort direction validation", err)
	}
}
