package txcos

import "testing"

var configTests = []struct {
	s   string
	cfg Config
}{
	{"cos:https://hostname:9999", Config{
		Endpoint:    "https://hostname:9999",
		Prefix:      "",
		Connections: 5,
		EnableCRC:   true,
	}},
	{"cos:https://hostname:9999/foobar", Config{
		Endpoint:    "https://hostname:9999",
		Prefix:      "foobar",
		Connections: 5,
		EnableCRC:   true,
	}},
	{"cos:https://hostname:9999/prefix/directory/", Config{
		Endpoint:    "https://hostname:9999",
		Prefix:      "prefix",
		Connections: 5,
		EnableCRC:   true,
	}},
	{"cos:hostname:9999/prefix/directory/", Config{
		Endpoint:    "hostname:9999",
		Prefix:      "prefix",
		Connections: 5,
		EnableCRC:   true,
	}},
}

func TestParseConfig(t *testing.T) {
	for i, test := range configTests {
		cfg, err := ParseConfig(test.s)
		if err != nil {
			t.Errorf("test %d:%s failed: %v", i, test.s, err)
			continue
		}

		if cfg != test.cfg {
			t.Errorf("test %d:\ninput:\n  %s\n wrong config, want:\n  %v\ngot:\n  %v",
				i, test.s, test.cfg, cfg)
			continue
		}
	}
}
