package config

// CoalesceString returns cliValue if non-empty, otherwise resolvedValue
// This is useful for merging CLI flags with resolved config values
func CoalesceString(cliValue, resolvedValue string) string {
	if cliValue != "" {
		return cliValue
	}
	return resolvedValue
}

// CoalesceInt returns cliValue if non-zero, otherwise resolvedValue
// This is useful for merging CLI flags with resolved config values
func CoalesceInt(cliValue, resolvedValue int) int {
	if cliValue != 0 {
		return cliValue
	}
	return resolvedValue
}

// CoalesceBool returns cliValue if condition is true, otherwise resolvedValue
// condition should indicate whether the CLI flag was explicitly set
// This is needed because bool's zero value (false) is a valid user choice
func CoalesceBool(cliValue, resolvedValue, condition bool) bool {
	if condition {
		return cliValue
	}
	return resolvedValue
}

// CoalesceMap merges CLI map with resolved map, with CLI values taking precedence
// This is useful for merging custom resources and other map-based configs
func CoalesceMap(cliMap, resolvedMap map[string]string) map[string]string {
	result := make(map[string]string)

	// Start with resolved map
	for k, v := range resolvedMap {
		result[k] = v
	}

	// Override with CLI map
	for k, v := range cliMap {
		result[k] = v
	}

	return result
}

// CoalesceStringSlice returns cliSlice if non-empty, otherwise resolvedSlice
// This is useful for merging CLI flags with resolved config values
func CoalesceStringSlice(cliSlice, resolvedSlice []string) []string {
	if len(cliSlice) > 0 {
		return cliSlice
	}
	return resolvedSlice
}
