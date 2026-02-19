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
		"kind":  reflect.TypeOf(sub).Elem().Name(),
		"sub":   children,
		"extra": sub.ExtraData(),
	}
}

func constJSONHandler(value any) http.Handler {
	resp, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	})
}

type stat struct {
	Name  string `json:"name"`
	Value uint64 `json:"value"`
}

func statsHandler(fn func() []stat) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fn())
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
