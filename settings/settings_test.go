package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustParseInt(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Value    string
		Expected int
	}{
		{Name: "positive", Value: "12", Expected: 12},
		{Name: "negative", Value: "-34", Expected: -34},
	} {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, MustParseInt(test.Value))
		})
	}

	assert.Panics(t, func() {
		MustParseInt("foo")
	})
}

func TestParseBool(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Value    string
		Expected bool
	}{
		{Name: "true", Value: "true", Expected: true},
		{Name: "upper cased true", Value: "TRUE", Expected: false},
		{Name: "false", Value: "false", Expected: false},
		{Name: "empty", Value: "", Expected: false},
		{Name: "other", Value: "foo", Expected: false},
	} {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, ParseBool(test.Value))
		})
	}
}
