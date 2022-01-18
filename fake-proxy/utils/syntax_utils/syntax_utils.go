package syntax_utils

func ConditionalInt(condition bool, onTrue, onFalse int) int {
	if condition {
		return onTrue
	} else {
		return onFalse
	}
}
