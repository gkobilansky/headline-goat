package dashboard

import "embed"

//go:embed assets/*
var Assets embed.FS

//go:embed templates/*
var Templates embed.FS
