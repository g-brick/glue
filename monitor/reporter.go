package monitor


// Reporter is an interface for gathering internal statistics.
type Reporter interface {
	// Statistics returns the statistics for the reporter,
	// with the given tags merged into the result.
	Statistics(tags map[string]string) []Statistic
}
