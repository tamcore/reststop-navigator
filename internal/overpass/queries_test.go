package overpass_test

import (
	"strings"
	"testing"

	"github.com/tamcore/reststop-navigator/internal/overpass"
)

func TestSupportedCountries(t *testing.T) {
	t.Parallel()
	got := overpass.SupportedCountries()
	want := map[overpass.CountryISO]bool{
		overpass.DE: true,
		overpass.AT: true,
		overpass.SK: true,
		overpass.CZ: true,
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for _, c := range got {
		if !want[c] {
			t.Errorf("unexpected country %q", c)
		}
	}
}

func TestIsSupported(t *testing.T) {
	t.Parallel()
	cases := map[overpass.CountryISO]bool{
		overpass.DE:               true,
		overpass.AT:               true,
		overpass.SK:               true,
		overpass.CZ:               true,
		overpass.CountryISO("FR"): false,
		overpass.CountryISO("XX"): false,
		overpass.CountryISO("de"): false,
		overpass.CountryISO(""):   false,
	}
	for c, want := range cases {
		if got := overpass.IsSupported(c); got != want {
			t.Errorf("IsSupported(%q) = %v, want %v", c, got, want)
		}
	}
}

func TestCountryQuery_Shape(t *testing.T) {
	t.Parallel()
	for _, c := range overpass.SupportedCountries() {
		c := c
		t.Run(string(c), func(t *testing.T) {
			t.Parallel()
			q, err := overpass.CountryQuery(c)
			if err != nil {
				t.Fatalf("CountryQuery(%q) error: %v", c, err)
			}
			must := []string{
				"[out:json]",
				`"ISO3166-1"`,
				string(c),
				"motorway",
				"trunk",
				"services",
				"rest_area",
				"fuel",
				"charging_station",
				"toilets",
				"restaurant",
				"out geom;",
			}
			for _, snippet := range must {
				if !strings.Contains(q, snippet) {
					t.Errorf("query for %q missing %q\nfull query:\n%s", c, snippet, q)
				}
			}
		})
	}
}

func TestCountryQuery_RejectsUnsupported(t *testing.T) {
	t.Parallel()
	cases := []overpass.CountryISO{"FR", "XX", "", "de"}
	for _, c := range cases {
		c := c
		t.Run(string(c), func(t *testing.T) {
			t.Parallel()
			if _, err := overpass.CountryQuery(c); err == nil {
				t.Errorf("CountryQuery(%q) unexpectedly succeeded", c)
			}
		})
	}
}
