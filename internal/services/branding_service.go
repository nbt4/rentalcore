package services

import commonbranding "github.com/nbt4/cores-common/pkg/branding"

// BrandingConfig re-exports the shared branding Config from cores-common.
type BrandingConfig = commonbranding.Config

// BrandingService re-exports the shared branding Service from cores-common.
type BrandingService = commonbranding.Service

// NewBrandingService creates a new branding service for the specified service name.
var NewBrandingService = commonbranding.NewService
