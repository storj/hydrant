package config

import (
	"bytes"
	"time"

	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/zeebo/errs/v2"
)

var (
	unmarshalOptions json.Options
	marshalOptions   json.Options
)

func init() {
	unmarshalOptions = json.WithUnmarshalers(
		json.UnmarshalFromFunc(UnmarshalSubmitter),
	)

	marshalOptions = json.JoinOptions(
		json.WithMarshalers(json.MarshalToFunc(MarshalSubmitter)),
		json.Deterministic(true),
		jsontext.EscapeForHTML(false),
		jsontext.EscapeForJS(false),
		jsontext.PreserveRawStrings(true),
	)
}

type Config struct {
	RefreshInterval time.Duration        `json:"refresh_interval,format:units"`
	Submitter       Submitter            `json:"submitter"`
	Submitters      map[string]Submitter `json:"submitters"`
}

// MarshalJSON implements the encoding/json Marshaler interface.
func (c Config) MarshalJSON() ([]byte, error) {
	type config Config // prevent recursion
	return json.Marshal(config(c), marshalOptions)
}

// UnmarshalJSON implements the encoding/json Unmarshaler interface.
func (c *Config) UnmarshalJSON(b []byte) error {
	type config Config // prevent recursion
	return json.Unmarshal(b, (*config)(c), unmarshalOptions)
}

type (
	Submitter interface{ isSubmitter() }

	MultiSubmitter []Submitter

	NamedSubmitter string

	FilterSubmitter struct {
		Filter    string    `json:"filter"`
		Submitter Submitter `json:"submitter"`
	}

	GrouperSubmitter struct {
		FlushInterval time.Duration `json:"flush_interval,format:units"`
		GroupBy       []string      `json:"group_by"`
		Submitter     Submitter     `json:"submitter"`
	}

	HTTPSubmitter struct {
		ProcessFields []string      `json:"process_fields"`
		Endpoint      string        `json:"endpoint"`
		FlushInterval time.Duration `json:"flush_interval,format:units"`
		MaxBatchSize  int           `json:"max_batch_size"`
	}

	OTelSubmitter struct {
		ProcessFields []string      `json:"process_fields"`
		Endpoint      string        `json:"endpoint"`
		FlushInterval time.Duration `json:"flush_interval,format:units"`
		MaxBatchSize  int           `json:"max_batch_size"`
	}

	PrometheusSubmitter struct {
		Namespace string    `json:"namespace"`
		Buckets   []float64 `json:"buckets"`
	}

	HydratorSubmitter struct {
	}

	TraceBufferSubmitter struct {
		BufferSize int `json:"buffer_size"`
	}

	NullSubmitter struct {
	}
)

func (MultiSubmitter) isSubmitter()      {}
func (NamedSubmitter) isSubmitter()      {}
func (FilterSubmitter) isSubmitter()     {}
func (GrouperSubmitter) isSubmitter()    {}
func (HTTPSubmitter) isSubmitter()       {}
func (OTelSubmitter) isSubmitter()       {}
func (PrometheusSubmitter) isSubmitter() {}
func (HydratorSubmitter) isSubmitter()      {}
func (TraceBufferSubmitter) isSubmitter()   {}
func (NullSubmitter) isSubmitter()          {}

//
// unmarshal support
//

func UnmarshalSubmitter(dec *jsontext.Decoder, dst *Submitter) error {
	raw, err := dec.ReadValue()
	if err != nil {
		return err
	}
	switch kind := raw.Kind(); kind {
	case '"':
		return unmarshalOneSubmitter[NamedSubmitter](raw, dst)

	case '[':
		return unmarshalOneSubmitter[MultiSubmitter](raw, dst)

	case '{':
		kind, err := findKind(raw)
		if err != nil {
			return err
		}
		switch kind.String() {
		case "filter":
			return unmarshalOneSubmitter[FilterSubmitter](raw, dst)

		case "grouper":
			return unmarshalOneSubmitter[GrouperSubmitter](raw, dst)

		case "http":
			return unmarshalOneSubmitter[HTTPSubmitter](raw, dst)

		case "otel":
			return unmarshalOneSubmitter[OTelSubmitter](raw, dst)

		case "prometheus":
			return unmarshalOneSubmitter[PrometheusSubmitter](raw, dst)

		case "hydrator":
			return unmarshalOneSubmitter[HydratorSubmitter](raw, dst)

		case "trace_buffer":
			return unmarshalOneSubmitter[TraceBufferSubmitter](raw, dst)

		case "null":
			return unmarshalOneSubmitter[NullSubmitter](raw, dst)

		default:
			return errs.Errorf("unknown submitter kind: %q", kind)
		}
	default:
		return errs.Errorf("unexpected json token kind: %q. expected string, object or list.", kind)
	}
}

func unmarshalOneSubmitter[T Submitter](data []byte, dst *Submitter) error {
	var into T
	if err := json.Unmarshal(data, &into, unmarshalOptions); err != nil {
		return err
	}
	*dst = into
	return nil
}

func findKind(data []byte) (jsontext.Token, error) {
	dec := jsontext.NewDecoder(bytes.NewBuffer(data),
		jsontext.AllowDuplicateNames(true),
	)

	tok, err := dec.ReadToken()
	if err != nil {
		return jsontext.Token{}, err
	} else if tok.Kind() != '{' {
		return jsontext.Token{}, errs.Errorf("expected object, got %v", tok.Kind())
	}

	for {
		tok, err := dec.ReadToken()
		if err != nil {
			return jsontext.Token{}, err
		} else if tok.Kind() == '}' {
			return jsontext.Token{}, errs.Errorf("missing kind field")
		} else if tok.Kind() != '"' {
			return jsontext.Token{}, errs.Errorf("unexpected token %q", tok)
		}

		if tok.String() != "kind" {
			if err := dec.SkipValue(); err != nil {
				return jsontext.Token{}, err
			}
			continue
		}

		tok, err = dec.ReadToken()
		if err != nil {
			return jsontext.Token{}, err
		} else if tok.Kind() != '"' {
			return jsontext.Token{}, errs.Errorf("expected string for kind, got %v", tok.Kind())
		}

		return tok, nil
	}
}

//
// marshal support
//

func MarshalSubmitter(enc *jsontext.Encoder, src Submitter) error {
	switch cfg := src.(type) {
	case *MultiSubmitter, *NamedSubmitter:
		return json.SkipFunc // the default is fine

	case *FilterSubmitter:
		type filterSubmitter FilterSubmitter // prevent recursion
		return marhsalOneSubmitter(enc, "filter", (*filterSubmitter)(cfg))

	case *GrouperSubmitter:
		type grouperSubmitter GrouperSubmitter // prevent recursion
		return marhsalOneSubmitter(enc, "grouper", (*grouperSubmitter)(cfg))

	case *HTTPSubmitter:
		type httpSubmitter HTTPSubmitter // prevent recursion
		return marhsalOneSubmitter(enc, "http", (*httpSubmitter)(cfg))

	case *OTelSubmitter:
		type otelSubmitter OTelSubmitter // prevent recursion
		return marhsalOneSubmitter(enc, "otel", (*otelSubmitter)(cfg))

	case *PrometheusSubmitter:
		type prometheusSubmitter PrometheusSubmitter // prevent recursion
		return marhsalOneSubmitter(enc, "prometheus", (*prometheusSubmitter)(cfg))

	case *HydratorSubmitter:
		type hydratorSubmitter HydratorSubmitter // prevent recursion
		return marhsalOneSubmitter(enc, "hydrator", (*hydratorSubmitter)(cfg))

	case *TraceBufferSubmitter:
		type traceBufferSubmitter TraceBufferSubmitter // prevent recursion
		return marhsalOneSubmitter(enc, "trace_buffer", (*traceBufferSubmitter)(cfg))

	case *NullSubmitter:
		type nullSubmitter NullSubmitter // prevent recursion
		return marhsalOneSubmitter(enc, "null", (*nullSubmitter)(cfg))

	default:
		return errs.Errorf("unknown submitter type %T", src)
	}
}

func marhsalOneSubmitter[T any](enc *jsontext.Encoder, kind string, src T) error {
	type kindedStruct[T any] struct {
		Kind string `json:"kind"`
		Sub  T      `json:",inline"`
	}
	return json.MarshalEncode(enc, kindedStruct[T]{kind, src}, marshalOptions)
}
