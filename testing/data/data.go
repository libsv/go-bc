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

//go:embed spv/binary/*
var spvBinaryData embed.FS

//go:embed spv/json/*
var spvSerialJSONData embed.FS

//go:embed spv/verify/*
var spvVerifyData embed.FS

//go:embed bhc/*
var blockHeaderData embed.FS

// SpvCreateData data for creating spv envelopes.
var SpvCreateData = TestDataDir{
	prefix: "spv/create",
	fs:     spvCreateData,
}

// SpvBinaryData data for creating spv envelopes.
var SpvBinaryData = TestDataDir{
	prefix: "spv/binary",
	fs:     spvBinaryData,
}

// SpvSerialJSONData data for creating spv envelopes.
var SpvSerialJSONData = TestDataDir{
	prefix: "spv/json",
	fs:     spvSerialJSONData,
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
