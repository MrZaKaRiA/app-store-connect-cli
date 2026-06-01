package ads

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/appleads"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

type adsReportPresetFlags struct {
	common commonFlags
	output shared.OutputFlags

	level           *string
	campaign        *string
	adGroup         *string
	from            *string
	to              *string
	lastDays        *int
	granularity     *string
	fields          *string
	sort            *string
	limit           *int
	offset          *int
	timeZone        *string
	returnRowTotals *bool
}

type adsReportPresetPayload struct {
	StartTime       string                  `json:"startTime"`
	EndTime         string                  `json:"endTime"`
	Granularity     string                  `json:"granularity,omitempty"`
	ReturnRowTotals bool                    `json:"returnRowTotals,omitempty"`
	Selector        adsReportPresetSelector `json:"selector"`
	TimeZone        string                  `json:"timeZone,omitempty"`
}

type adsReportPresetSelector struct {
	Fields     []string                   `json:"fields,omitempty"`
	OrderBy    []adsReportPresetSort      `json:"orderBy,omitempty"`
	Pagination *adsReportPresetPagination `json:"pagination,omitempty"`
}

type adsReportPresetSort struct {
	Field     string `json:"field"`
	SortOrder string `json:"sortOrder"`
}

type adsReportPresetPagination struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type adsReportLevelSpec struct {
	commandPath []string
}

var adsReportLevels = map[string]adsReportLevelSpec{
	"campaigns":             {commandPath: []string{"reports", "campaigns"}},
	"ad-groups":             {commandPath: []string{"reports", "ad-groups"}},
	"keywords":              {commandPath: []string{"reports", "keywords"}},
	"search-terms":          {commandPath: []string{"reports", "search-terms"}},
	"ads":                   {commandPath: []string{"reports", "ads"}},
	"ad-group-keywords":     {commandPath: []string{"reports", "ad-group-keywords"}},
	"ad-group-search-terms": {commandPath: []string{"reports", "ad-group-search-terms"}},
}

// ReportsPresetCommand returns an operator-friendly Apple Ads reporting helper.
func ReportsPresetCommand() *ffcli.Command {
	fs := flag.NewFlagSet("preset", flag.ExitOnError)
	flags := adsReportPresetFlags{
		common: commonFlags{
			AdsProfile: fs.String("ads-profile", "", "Use named Apple Ads authentication profile"),
			Org:        fs.String("org", "", "Apple Ads organization ID (or ASC_ADS_ORG_ID env)"),
		},
		output:          shared.BindOutputFlags(fs),
		level:           fs.String("level", "campaigns", "Report level: campaigns, ad-groups, keywords, search-terms, ads, ad-group-keywords, ad-group-search-terms"),
		campaign:        fs.String("campaign", "", "Campaign ID for campaign-scoped report levels"),
		adGroup:         fs.String("ad-group", "", "Ad group ID for ad-group-scoped report levels"),
		from:            fs.String("from", "", "Start date in YYYY-MM-DD"),
		to:              fs.String("to", "", "End date in YYYY-MM-DD"),
		lastDays:        fs.Int("last-days", 0, "Use an inclusive range ending today in --time-zone"),
		granularity:     fs.String("granularity", "DAILY", "Report granularity: DAILY, WEEKLY, MONTHLY"),
		fields:          fs.String("fields", "", "Comma-separated selector fields to request"),
		sort:            fs.String("sort", "", "Sort field with optional direction, e.g. impressions:desc"),
		limit:           fs.Int("limit", 1000, "Report row limit (1..1000)"),
		offset:          fs.Int("offset", 0, "Report row offset (>=0)"),
		timeZone:        fs.String("time-zone", "UTC", "IANA reporting time zone"),
		returnRowTotals: fs.Bool("return-row-totals", false, "Request row totals in the report response"),
	}

	return &ffcli.Command{
		Name:       "preset",
		ShortUsage: "asc ads reports preset --level campaigns --from YYYY-MM-DD --to YYYY-MM-DD [flags]",
		ShortHelp:  "Build and run Apple Ads report presets without JSON payloads.",
		LongHelp: `Build and run Apple Ads report presets without JSON payloads.

This helper builds a documented ReportingRequest for the existing Apple Ads
report endpoints. Choose the report resource with --level. Campaign-scoped and
ad-group-scoped report levels require --campaign and/or --ad-group.

Use --last-days for an inclusive rolling date range; "today" is calculated in
--time-zone. Use the raw report commands with --file when you need custom
conditions or advanced selector JSON. Ad-level reports require --sort because
Apple Ads requires selector.orderBy for that endpoint.

Examples:
  asc ads reports preset --level campaigns --from 2026-05-01 --to 2026-05-31 --fields campaignName,impressions,taps,spend --sort impressions:desc --org "123456"
  asc ads reports preset --level keywords --campaign 12345 --last-days 7 --time-zone America/Los_Angeles --fields keyword,impressions,taps --org "123456"
  asc ads reports preset --level ads --campaign 12345 --from 2026-05-01 --to 2026-05-31 --sort impressions:desc --org "123456"`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Exec: func(ctx context.Context, args []string) error {
			if err := rejectUnexpectedArgs(args); err != nil {
				return err
			}
			return executeReportsPreset(ctx, flags)
		},
	}
}

func executeReportsPreset(ctx context.Context, flags adsReportPresetFlags) error {
	level := strings.TrimSpace(*flags.level)
	levelSpec, ok := adsReportLevels[level]
	if !ok {
		return shared.UsageError("--level must be one of: " + strings.Join(sortedReportPresetLevels(), ", "))
	}
	spec, ok := appleads.EndpointByCommandPath(levelSpec.commandPath...)
	if !ok {
		return fmt.Errorf("ads reports preset: endpoint for level %q is not registered", level)
	}
	pathParams, err := reportPresetPathParams(spec, flags)
	if err != nil {
		return shared.UsageError(err.Error())
	}
	payload, err := buildReportPresetPayload(flags, time.Now().UTC())
	if err != nil {
		return shared.UsageError(err.Error())
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("ads reports preset: marshal request: %w", err)
	}

	client, err := resolveClient(ctx, flags.common, spec.RequiresOrg)
	if err != nil {
		return fmt.Errorf("ads: %w", err)
	}

	requestCtx, cancel := requestContext(ctx)
	defer cancel()

	result, err := client.Do(requestCtx, spec, pathParams, url.Values{}, body)
	if err != nil {
		return fmt.Errorf("ads reports preset: %w", err)
	}
	return shared.PrintOutput(result, *flags.output.Output, *flags.output.Pretty)
}

func reportPresetPathParams(spec appleads.EndpointSpec, flags adsReportPresetFlags) (map[string]string, error) {
	params := map[string]string{}
	for _, param := range spec.PathParams {
		var raw string
		switch param.Name {
		case "campaignId":
			raw = strings.TrimSpace(*flags.campaign)
		case "adgroupId":
			raw = strings.TrimSpace(*flags.adGroup)
		default:
			return nil, fmt.Errorf("unsupported report path parameter %q", param.Name)
		}
		if raw == "" {
			return nil, fmt.Errorf("--%s is required for --level %s", param.Flag, strings.TrimSpace(*flags.level))
		}
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("--%s must be an integer", param.Flag)
		}
		if parsed < 0 {
			return nil, fmt.Errorf("--%s must be >= 0", param.Flag)
		}
		params[param.Name] = raw
	}
	return params, nil
}

func buildReportPresetPayload(flags adsReportPresetFlags, now time.Time) (adsReportPresetPayload, error) {
	reportingTimeZone := strings.TrimSpace(*flags.timeZone)
	reportingLocation, err := reportPresetLocation(reportingTimeZone)
	if err != nil {
		return adsReportPresetPayload{}, err
	}
	start, end, err := reportPresetDateRange(*flags.from, *flags.to, *flags.lastDays, now, reportingLocation)
	if err != nil {
		return adsReportPresetPayload{}, err
	}
	granularity := strings.ToUpper(strings.TrimSpace(*flags.granularity))
	if granularity == "" {
		return adsReportPresetPayload{}, fmt.Errorf("--granularity is required")
	}
	if !slices.Contains([]string{"DAILY", "WEEKLY", "MONTHLY"}, granularity) {
		return adsReportPresetPayload{}, fmt.Errorf("--granularity must be one of: DAILY, WEEKLY, MONTHLY")
	}
	if *flags.limit < 1 || *flags.limit > appleads.MaxPageLimit(appleads.EndpointSpec{}) {
		return adsReportPresetPayload{}, fmt.Errorf("--limit must be between 1 and 1000")
	}
	if *flags.offset < 0 {
		return adsReportPresetPayload{}, fmt.Errorf("--offset must be >= 0")
	}
	if strings.TrimSpace(*flags.level) == "ads" && strings.TrimSpace(*flags.sort) == "" {
		return adsReportPresetPayload{}, fmt.Errorf("--sort is required for --level ads")
	}

	selector := adsReportPresetSelector{
		Fields: shared.SplitCSV(*flags.fields),
		Pagination: &adsReportPresetPagination{
			Offset: *flags.offset,
			Limit:  *flags.limit,
		},
	}
	if sortValue := strings.TrimSpace(*flags.sort); sortValue != "" {
		sortSpec, err := parseReportPresetSort(sortValue)
		if err != nil {
			return adsReportPresetPayload{}, err
		}
		selector.OrderBy = []adsReportPresetSort{sortSpec}
	}
	return adsReportPresetPayload{
		StartTime:       start,
		EndTime:         end,
		Granularity:     granularity,
		ReturnRowTotals: *flags.returnRowTotals,
		Selector:        selector,
		TimeZone:        reportingTimeZone,
	}, nil
}

func reportPresetDateRange(from, to string, lastDays int, now time.Time, reportingLocation *time.Location) (string, string, error) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if lastDays < 0 {
		return "", "", fmt.Errorf("--last-days must be >= 0")
	}
	if lastDays > 0 {
		if from != "" || to != "" {
			return "", "", fmt.Errorf("--last-days cannot be combined with --from or --to")
		}
		reportingNow := now.In(reportingLocation)
		end := reportingNow.Format("2006-01-02")
		start := reportingNow.AddDate(0, 0, -(lastDays - 1)).Format("2006-01-02")
		return start, end, nil
	}
	if from == "" || to == "" {
		return "", "", fmt.Errorf("either --last-days or both --from and --to are required")
	}
	startDate, err := parseReportPresetDate("--from", from)
	if err != nil {
		return "", "", err
	}
	endDate, err := parseReportPresetDate("--to", to)
	if err != nil {
		return "", "", err
	}
	if endDate.Before(startDate) {
		return "", "", fmt.Errorf("--to must be on or after --from")
	}
	return from, to, nil
}

func reportPresetLocation(value string) (*time.Location, error) {
	if value == "" {
		return time.UTC, nil
	}
	location, err := time.LoadLocation(value)
	if err != nil {
		return nil, fmt.Errorf("--time-zone must be a valid IANA time zone")
	}
	return location, nil
}

func parseReportPresetDate(flagName string, value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be in YYYY-MM-DD format", flagName)
	}
	return parsed, nil
}

func parseReportPresetSort(value string) (adsReportPresetSort, error) {
	field, direction, ok := strings.Cut(value, ":")
	field = strings.TrimSpace(field)
	if field == "" {
		return adsReportPresetSort{}, fmt.Errorf("--sort field is required")
	}
	sortOrder := "DESCENDING"
	if ok {
		switch strings.ToLower(strings.TrimSpace(direction)) {
		case "asc", "ascending":
			sortOrder = "ASCENDING"
		case "desc", "descending":
			sortOrder = "DESCENDING"
		default:
			return adsReportPresetSort{}, fmt.Errorf("--sort direction must be asc or desc")
		}
	}
	return adsReportPresetSort{Field: field, SortOrder: sortOrder}, nil
}

func sortedReportPresetLevels() []string {
	levels := make([]string, 0, len(adsReportLevels))
	for level := range adsReportLevels {
		levels = append(levels, level)
	}
	slices.Sort(levels)
	return levels
}
