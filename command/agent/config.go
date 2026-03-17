package main

import "go.scnd.dev/open/polygon"

type Config struct {
	AppName               *string   `yaml:"appName" validate:"omitempty"`
	Address               *string   `yaml:"address" validate:"required"`
	Secret                *string   `yaml:"secret" validate:"required"`
	TelemetryUrl          *string   `yaml:"telemetryUrl" validate:"omitempty"`
	TelemetryOrganization *string   `yaml:"telemetryOrganization" validate:"omitempty"`
	Zones                 []*string `yaml:"zones" validate:"required"`
	Upstream              *string   `yaml:"upstream" validate:"omitempty"`
	RecordKey             *string   `yaml:"recordKey" validate:"omitempty"`
	RecordFile            *string   `yaml:"recordFile" validate:"omitempty"`
	WebListen             []*string `yaml:"webListen" validate:"omitempty"`
	CertificateFile       *string   `yaml:"certificateFile" validate:"omitempty"`
}

func (r *Config) GetRecordKey() *string {
	return r.RecordKey
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
