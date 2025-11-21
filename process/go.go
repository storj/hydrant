package process

import (
	"runtime"
	"runtime/debug"
	"time"

	"storj.io/hydrant"
)

// MustRegisterRuntimeAnnotations is run on the default store at init.
func MustRegisterRuntimeAnnotations(s *Store) {
	s.MustRegisterAnnotation(
		hydrant.String("go.os", runtime.GOOS),
		hydrant.String("go.arch", runtime.GOARCH),
	)

	if info, ok := debug.ReadBuildInfo(); ok {
		s.MustRegisterAnnotation(
			hydrant.String("go.version", info.GoVersion),
			hydrant.String("go.main.path", info.Main.Path),
			hydrant.String("go.main.version", info.Main.Version),
			hydrant.String("go.main.sum", info.Main.Sum),
		)
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.time":
				if ts, err := time.Parse(time.RFC3339, setting.Value); err == nil {
					s.MustRegisterAnnotation(hydrant.Timestamp("go.vcs.time", ts))
				}
			case "vcs.revision":
				s.MustRegisterAnnotation(hydrant.String("go.vcs.rev", setting.Value))
			case "vcs.modified":
				s.MustRegisterAnnotation(hydrant.Bool("go.vcs.modified", setting.Value != "false"))
			}
		}
	}
}

func init() {
	MustRegisterRuntimeAnnotations(DefaultStore)
}
