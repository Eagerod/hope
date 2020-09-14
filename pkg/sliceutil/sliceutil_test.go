package sliceutil

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestStringInSlice(t *testing.T) {
	var tests = []struct {
		name string
		ret  bool
		val  string
		sli  []string
	}{
		{"Contains", true, "something", []string{"anything", "in", "something", "else"}},
		{"Contains End", true, "anything", []string{"anything", "in", "something", "else"}},
		{"Doesn't Contain", false, "anyone", []string{"anything", "in", "something", "else"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.ret, StringInSlice(tt.val, tt.sli))
		})
	}
}
