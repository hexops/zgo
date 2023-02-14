package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/hexops/zgo/internal/errors"
)

type Config struct {
	// Version is the Zig version to use, e.g. "0.11.0-dev.1615+f62e3b8c0" from https://ziglang.org/download/
	//
	// If set to "system", then the system's zig installation on the PATH is used.
	//
	// If not set, the version in the .zgo/ Dir is used, otherwise the latest nightly version is
	// fetched and written to the .zgo/ Dir.
	//
	// Alternatively specified using ZGO_VERSION
	Version string `toml:"version"`

	// Whether or not to print verbose zgo information
	//
	// Alternatively specified using ZGO_VERBOSE
	Verbose bool `toml:"verbose"`

	// Dir is where zgo should download dependencies like the minimal macOS SDK, or Zig binary to.
	//
	// Defaults to ".zgo"
	//
	// Alternatively specified using ZGO_DIR
	Dir string `toml:"dir"`

	// The macOS SDK is distributed under the terms of the Xcode and Apple SDKs agreement:
	//
	// https://www.apple.com/legal/sla/docs/xcode.pdf
	//
	// You must agree to those terms before downloading.
	//
	// Alternatively specified using ZGO_ACCEPT_XCODE_LIENSE
	AcceptXCodeLicense bool `toml:"acceptXCodeLicense"`
}

func LoadConfig(file string, out *Config) error {
	_, err := toml.DecodeFile(file, out)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Default config values.
	if out.Version == "" {
		out.Version = os.Getenv("ZGO_VERSION")
		if out.Version == "" {
			latestNightly, err := queryLatestNightlyZigVersion()
			if err != nil {
				return errors.Wrap(err, "querying latest nightly Zig version")
			}
			out.Version = latestNightly
		}
	}
	if !out.Verbose {
		out.Verbose, _ = strconv.ParseBool(os.Getenv("ZGO_VERBOSE"))
	}
	if !out.AcceptXCodeLicense {
		out.AcceptXCodeLicense, _ = strconv.ParseBool(os.Getenv("ZGO_ACCEPT_XCODE_LICENSE"))
	}
	if out.Dir == "" {
		out.Dir = os.Getenv("ZGO_DIR")
		if out.Dir == "" {
			out.Dir = ".zgo"
		}
	}
	return nil
}

func queryLatestNightlyZigVersion() (string, error) {
	resp, err := http.Get("https://ziglang.org/download/index.json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var v *struct {
		Master struct {
			Version string
		}
	} = nil
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", err
	}
	return v.Master.Version, nil
}
