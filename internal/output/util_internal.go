package output

import "strconv"

func intToString(i int) string { return strconv.Itoa(i) }
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
