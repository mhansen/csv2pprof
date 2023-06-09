package main

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/pprof/profile"
)

func TestComments(t *testing.T) {
	p, err := ConvertCSVToPprof(strings.NewReader("stack,samples/count,time/ms\nfoo;bar,1,1000"))
	if err != nil {
		t.Errorf("got error: %v", err)
	}

	wantComments := []string{"Generated by csv2pprof"}
	if diff := cmp.Diff(wantComments, p.Comments); diff != "" {
		t.Errorf("wanted comments %v got %v, diff: %v", wantComments, p.Comments, diff)
	}
}

func TestUnits(t *testing.T) {
	type test struct {
		input string
		want  []*profile.ValueType
	}

	tests := []test{
		{
			// Two units fully specified
			input: "stack,samples/count,time/ms\nfoo;bar,1,1000",
			want: []*profile.ValueType{
				{
					Type: "samples",
					Unit: "count",
				},
				{
					Type: "time",
					Unit: "ms",
				},
			},
		},
		{
			// One unit fully specified
			input: "stack,time/ms\nfoo;bar,1",
			want: []*profile.ValueType{
				{
					Type: "time",
					Unit: "ms",
				},
			},
		},
		{
			// No units given, default to 'count'
			input: "stack,samples\nfoo;bar,1",
			want: []*profile.ValueType{
				{
					Type: "samples",
					Unit: "count",
				},
			},
		},
		{
			// If multiple possible units, choose the last one.
			input: "stack,samples/unit1/unit2\nfoo;bar,1",
			want: []*profile.ValueType{
				{
					Type: "samples/unit1",
					Unit: "unit2",
				},
			},
		},
		{
			// Stack at the end
			input: "time/seconds,stack\n1,foo;bar",
			want: []*profile.ValueType{
				{
					Type: "time",
					Unit: "seconds",
				},
			},
		},
		{
			// Stack in the middle
			input: "time/seconds,stack,age/years\n1,foo;bar,18",
			want: []*profile.ValueType{
				{
					Type: "time",
					Unit: "seconds",
				},
				{
					Type: "age",
					Unit: "years",
				},
			},
		},
	}

	for i, c := range tests {
		p, err := ConvertCSVToPprof(strings.NewReader(c.input))
		if err != nil {
			t.Errorf("got error: %v", err)
		}

		opts := cmpopts.IgnoreUnexported(profile.ValueType{})
		got := p.SampleType
		if diff := cmp.Diff(c.want, got, opts); diff != "" {
			t.Errorf("test %v, wanted SampleType %#v got %#v, diff: %v", i, c.want, got, diff)
		}
	}
}

func TestSamples(t *testing.T) {
	type test struct {
		input string
		want  []*profile.Sample
	}

	tests := []test{
		{
			// Stack at the start
			input: "stack,samples/count\nfoo;bar,1",
			want: []*profile.Sample{
				{Value: []int64{1}},
			},
		},
		{
			// Stack at the end
			input: "samples/count,stack\n1,foo;bar",
			want: []*profile.Sample{
				{Value: []int64{1}},
			},
		},
		{
			// Stack in the middle end
			input: "samples/count,stack,age/years\n1,foo;bar,18",
			want: []*profile.Sample{
				{Value: []int64{1, 18}},
			},
		},
		{
			input: "stack,samples/count,time/ms\nfoo;bar,1,1000",
			want: []*profile.Sample{
				{Value: []int64{1, 1000}},
			},
		},
		{
			input: "stack,samples/count,time/ms\nfoo;bar,1,1000\nfoo,2,2000",
			want: []*profile.Sample{
				{Value: []int64{1, 1000}},
				{Value: []int64{2, 2000}},
			},
		},
	}

	for _, c := range tests {
		p, err := ConvertCSVToPprof(strings.NewReader(c.input))
		if err != nil {
			t.Errorf("got error: %v", err)
		}

		if len(p.Sample) != len(c.want) {
			t.Errorf("wanted %v samples got %v samples. samples: %v", len(c.want), len(c.want), c.want)
			continue
		}
		got := p.Sample
		opts := []cmp.Option{
			cmpopts.IgnoreUnexported(profile.Sample{}),
			cmpopts.IgnoreFields(profile.Sample{}, "Location"),
		}
		if diff := cmp.Diff(c.want, got, opts...); diff != "" {
			t.Errorf("wanted Sample %#v got %#v diff: %v", c.want, got, diff)
		}
	}
}

func TestErrors(t *testing.T) {
	type test struct {
		input   string
		wantErr string
	}

	tests := []test{
		{
			input:   "samples/count\n1",
			wantErr: "expected \"stack\" in CSV header row, got: [\"samples/count\"]",
		},
		{
			input:   "stack\nfoo;bar",
			wantErr: "expected columns with weights in CSV header row, got [\"stack\"]",
		},
		{
			input:   "stack,weight\nfoo;bar",
			wantErr: "error reading CSV: record on line 2: wrong number of fields",
		},
		{
			input:   "stack,weight\nfoo;bar,not-a-number",
			wantErr: "on line 2, couldn't parse number: strconv.ParseInt: parsing \"not-a-number\": invalid syntax",
		},
	}
	for i, c := range tests {
		_, err := ConvertCSVToPprof(strings.NewReader(c.input))
		if err == nil {
			t.Errorf("test %v, wanted error %q, got error nil", i, c.wantErr)
			continue
		}
		if err.Error() != c.wantErr {
			t.Errorf("test %v, wanted error %q, got error: %q", i, c.wantErr, err)
		}
	}
}
