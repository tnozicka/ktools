package version

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"time"

	"k8s.io/klog/v2"
)

func OptionalToString[T any](ptr *T) string {
	if ptr == nil {
		return "<unknown>"
	}

	return fmt.Sprintf("%v", *ptr)
}

type Info struct {
	Revision     *string
	RevisionTime *time.Time
	Modified     *bool
	GoVersion    *string
}

func (i *Info) String() string {
	return fmt.Sprintf(
		"Revision: %v, RevisionTime: %v, Modified: %v, GoVersion: %v",
		OptionalToString(i.Revision),
		OptionalToString(i.RevisionTime),
		OptionalToString(i.Modified),
		OptionalToString(i.GoVersion),
	)
}

func Get() *Info {
	info := &Info{}

	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		klog.ErrorS(fmt.Errorf("can't read build info"), "Unable to determine version")
		return &Info{}
	}

	info.GoVersion = new(buildInfo.GoVersion)

	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.revision":
			info.Revision = new(setting.Value)

		case "vcs.time":
			t, err := time.Parse(time.RFC3339, setting.Value)
			if err != nil {
				klog.ErrorS(err, "Can't to determine version")
			} else {
				info.RevisionTime = new(t)
			}

		case "vcs.modified":
			b, err := strconv.ParseBool(setting.Value)
			if err != nil {
				klog.ErrorS(err, "Can't determine vcs state")
			} else {
				info.Modified = new(b)
			}
		}
	}

	return info
}
