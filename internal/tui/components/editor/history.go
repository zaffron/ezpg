package editor

func NewHistory() *History {
	return &History{
		pos: -1,
	}
}
