package extract

import "regexp"

// readsPattern matches the game files a build may read: the tables in Data (not the
// translations under Data/<language>/), object templates and stat descriptions. It is wider than
// what Dataset reads today, so adding a table or template needs no new download rule.
var readsPattern = regexp.MustCompile(`(?i)^(data/[^/]+\.datc64|metadata/.*\.(ot|it|otc)|metadata/statdescriptions/.*)$`)

// Reads reports whether a build may read the game file at path.
func Reads(path string) bool { return readsPattern.MatchString(path) }
