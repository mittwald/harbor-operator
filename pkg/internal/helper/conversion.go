package helper

// BoolToString returns a string depending on the given boolean value
func BoolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
