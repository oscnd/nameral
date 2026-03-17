package config

import (
	"go.scnd.dev/open/polygon"
)

type Config struct {
	AppName               *string   `yaml:"appName" validate:"required"`
	WebListen             []*string `yaml:"webListen" validate:"required"`
	ProtoListen           []*string `yaml:"protoListen" validate:"required"`
	DnsListen             *string   `yaml:"dnsListen" validate:"required"`
	TelemetryUrl          *string   `yaml:"telemetryUrl" validate:"required"`
	TelemetryOrganization *string   `yaml:"telemetryOrganization" validate:"omitempty"`
	RedisAddress          *string   `yaml:"redisAddress" validate:"required"`
	RedisPassword         *string   `yaml:"redisPassword" validate:"omitempty"`
	RedisDatabase         *int      `yaml:"redisDatabase" validate:"omitempty"`
	ServerCertificateFile *string   `yaml:"serverCertificateFile" validate:"required"`
	ServerPrivateKeyFile  *string   `yaml:"serverPrivateKeyFile" validate:"required"`
	DnssecPath            *string   `yaml:"dnssecPath" validate:"omitempty"`
	DnssecZones           []*string `yaml:"dnssecZones" validate:"omitempty"`
	Clients               []*Client `yaml:"clients" validate:"required,dive"`
}

type Client struct {
	Name         *string   `yaml:"name" validate:"required"`
	Token        *string   `yaml:"token" validate:"required"`
	AllowedZones []*string `yaml:"allowedZones" validate:"required"`
}

func (r *Config) GetWebListen() []*string {
	return r.WebListen
}

func (r *Config) GetRedisAddress() *string {
	return r.RedisAddress
}

func (r *Config) GetRedisPassword() *string {
	return r.RedisPassword
}

func (r *Config) GetRedisDatabase() *int {
	return r.RedisDatabase
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
