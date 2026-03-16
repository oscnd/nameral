package config

import (
	"go.scnd.dev/open/polygon"
)

type Config struct {
	AppName               *string   `yaml:"appName" validate:"required"`
	ProtoListen           []*string `yaml:"protoListen" validate:"required"`
	DnsListen             *string   `yaml:"dnsListen" validate:"required"`
	TelemetryUrl          *string   `yaml:"telemetryUrl" validate:"required"`
	TelemetryOrganization *string   `yaml:"telemetryOrganization" validate:"omitempty"`
}

func (r *Config) GetProtoListen() []*string {
	return r.ProtoListen
}

func (r *Config) GetPolygonConfig() *polygon.Config {
	return &polygon.Config{
		AppName:               r.AppName,
		AppVersion:            nil,
		AppNamespace:          nil,
		AppInstanceId:         nil,
		TelemetryUrl:          r.TelemetryUrl,
		TelemetryOrganization: r.TelemetryOrganization,
	}
}
