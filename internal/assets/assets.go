package assets

import "embed"

//go:embed regex/*
var EmbeddedRegexFS embed.FS
