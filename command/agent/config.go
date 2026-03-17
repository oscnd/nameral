package main

type Config struct {
	Address         *string   `yaml:"address" validate:"required"`
	Secret          *string   `yaml:"secret" validate:"required"`
	Zones           []*string `yaml:"zones" validate:"required"`
	Upstream        *string   `yaml:"upstream" validate:"required"`
	CertificateFile *string   `yaml:"certificateFile" validate:"omitempty"`
}
