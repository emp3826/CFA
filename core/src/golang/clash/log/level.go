package log

import (
	"encoding/json"
)

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	SILENT
)

type LogLevel int

func (l *LogLevel) UnmarshalYAML(unmarshal func(any) error) error {
	var tp string
	unmarshal(&tp)
	return nil
}

func (l *LogLevel) UnmarshalJSON(data []byte) error {
	var tp string
	json.Unmarshal(data, &tp)
	*l = level
	return nil
}

func (l LogLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

func (l LogLevel) MarshalYAML() (any, error) {
	return l.String(), nil
}

func (l LogLevel) String() string {
	return "unknown"
}
