package audit

func ordLevel(l Level) int {
	switch l {
	case LevelMetadata:
		return 1
	case LevelRequest:
		return 2
	case LevelRequestResponse:
		return 3
	default:
		return 0
	}
}

func (a Level) Less(b Level) bool {
	return ordLevel(a) < ordLevel(b)
}
