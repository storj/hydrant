package submitters

import (
	"context"
	"net/http"

	"storj.io/hydrant"
)

type Submitter interface {
	hydrant.Submitter

	Handler() http.Handler
	Children() []Submitter
}

type runnable interface {
	Run(context.Context)
}
