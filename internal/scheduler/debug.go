package scheduler

import (
	"os"
	"strings"
)

func schedulerDebugEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("SCHEDULER_DEBUG")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func schedulerTestModeEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("SCHEDULER_TEST_MODE")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
