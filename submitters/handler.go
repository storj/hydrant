package submitters

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
)

func treeify(sub Submitter) any {
	if sub, ok := sub.(*lateSubmitter); ok {
		return treeify(sub.sub)
	}
	var children []any
	for _, child := range sub.Children() {
		children = append(children, treeify(child))
	}
	return map[string]any{
		"kind": reflect.TypeOf(sub).Elem().Name(),
		"sub":  children,
	}
}

func constJSONHandler(value any) http.Handler {
	resp, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(resp)
	})
}

func parseBool(s string, def bool) bool {
	x, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return x
}

func parseInt(s string, def int) int {
	x, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return x
}
