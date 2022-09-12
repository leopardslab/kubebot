package otlp

// DefaultTickTime will be used when there's enableDebugTimer is true but tickTime is not specified, the default value is set to 60 seconds
const (
	DefaultTickTime = 60
)
var (
	LogCounter = 0
)

func getTickTime(tickTime int) int {
	if tickTime == 0 {
		return DefaultTickTime
	} else {
		return tickTime
	}
}
