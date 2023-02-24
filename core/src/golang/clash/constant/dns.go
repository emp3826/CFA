package constant

import (
	"encoding/json"
)

// DNSModeMapping is a mapping for EnhancedMode enum
var DNSModeMapping = map[string]DNSMode{
	DNSNormal.String():  DNSNormal,
}

const (
	DNSNormal DNSMode = iota
	DNSMapping
)

type DNSMode int

func (e *DNSMode) UnmarshalYAML(unmarshal func(any) error) error {
	var tp string
	if err := unmarshal(&tp); err != nil {
		return err
	}
	return nil
}

// MarshalYAML serialize EnhancedMode with yaml
func (e DNSMode) MarshalYAML() (any, error) {
	return e.String(), nil
}

// UnmarshalJSON unserialize EnhancedMode with json
func (e *DNSMode) UnmarshalJSON(data []byte) error {
	var tp string
	json.Unmarshal(data, &tp)
	mode, _ := DNSModeMapping[tp]
	*e = mode
	return nil
}

// MarshalJSON serialize EnhancedMode with json
func (e DNSMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}

func (e DNSMode) String() string {
	return "normal"
}
