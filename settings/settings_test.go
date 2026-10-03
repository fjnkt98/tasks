package settings

import "testing"

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
			if actual := MustParseInt(test.Value); actual != test.Expected {
				t.Errorf("expected %v, but got %v", test.Expected, actual)
			}
		})
	}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	MustParseInt("foo")
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
			if actual := ParseBool(test.Value); actual != test.Expected {
				t.Errorf("expected %v, but got %v", test.Expected, actual)
			}
		})
	}
}
