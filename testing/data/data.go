package data

import (
	"embed"
	"path"
)

// TestDataDir a directory container test data.
type TestDataDir struct {
	prefix string
	fs     embed.FS
}

//go:embed spv/create/*
var spvCreateData embed.FS

//go:embed spv/verify/*
var spvVerifyData embed.FS

//go:embed bhc/*
var blockHeaderData embed.FS

// SpvCreateData data for creating spv envelopes.
var SpvCreateData = TestDataDir{
	prefix: "spv/create",
	fs:     spvCreateData,
}

// SpvVerifyData data for verifying spv envelopes.
var SpvVerifyData = TestDataDir{
	prefix: "spv/verify",
	fs:     spvVerifyData,
}

// BlockHeaderData hash => block header mapping data.
var BlockHeaderData = TestDataDir{
	prefix: "bhc",
	fs:     blockHeaderData,
}

// Load the data of a file.
func (d *TestDataDir) Load(file string) ([]byte, error) {
	return d.fs.ReadFile(path.Join(d.prefix, file))
}
