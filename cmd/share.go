/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

const (
	// Default expiry is 7 days (168h).
	shareDefaultExpiry = time.Duration(604800) * time.Second
)

// Upload specific flags.
var (
	shareFlagContentType = cli.StringFlag{
		Name:  "content-type, T",
		Usage: "specify a content-type to allow",
	}
	shareFlagExpire = cli.StringFlag{
		Name:  "expire, E",
		Value: "168h",
		Usage: "set expiry in NN[h|m|s]",
	}
)

// Structured share command message.
type shareMesssage struct {
	Status      string        `json:"status"`
	ObjectURL   string        `json:"url"`
	ShareURL    string        `json:"share"`
	TimeLeft    time.Duration `json:"timeLeft"`
	ContentType string        `json:"contentType,omitempty"` // Only used by upload cmd.
}

// String - Themefied string message for console printing.
func (s shareMesssage) String() string {
	msg := console.Colorize("URL", fmt.Sprintf("URL: %s\n", s.ObjectURL))
	msg += console.Colorize("Expire", fmt.Sprintf("Expire: %s\n", timeDurationToHumanizedDuration(s.TimeLeft)))
	if s.ContentType != "" {
		msg += console.Colorize("Content-type", fmt.Sprintf("Content-Type: %s\n", s.ContentType))
	}

	// Highlight <FILE> specifically. "share upload" sub-commands use this identifier.
	shareURL := strings.Replace(s.ShareURL, "<FILE>", console.Colorize("File", "<FILE>"), 1)
	// Highlight <KEY> specifically for recursive operation.
	shareURL = strings.Replace(shareURL, "<NAME>", console.Colorize("File", "<NAME>"), 1)

	msg += console.Colorize("Share", fmt.Sprintf("Share: %s\n", shareURL))

	return msg
}

// JSON - JSONified message for scripting.
func (s shareMesssage) JSON() string {
	s.Status = "success"
	shareMessageBytes, e := json.Marshal(s)
	fatalIf(probe.NewError(e), "Failed to marshal into JSON.")

	// JSON encoding escapes ampersand into its unicode character
	// which is not usable directly for share and fails with cloud
	// storage. convert them back so that they are usable.
	shareMessageBytes = bytes.Replace(shareMessageBytes, []byte("\\u0026"), []byte("&"), -1)
	shareMessageBytes = bytes.Replace(shareMessageBytes, []byte("\\u003c"), []byte("<"), -1)
	shareMessageBytes = bytes.Replace(shareMessageBytes, []byte("\\u003e"), []byte(">"), -1)

	return string(shareMessageBytes)
}

// shareSetColor sets colors share sub-commands.
func shareSetColor() {
	// Additional command speific theme customization.
	console.SetColor("URL", color.New(color.Bold))
	console.SetColor("Expire", color.New(color.FgCyan))
	console.SetColor("Content-type", color.New(color.FgBlue))
	console.SetColor("Share", color.New(color.FgGreen))
	console.SetColor("File", color.New(color.FgRed, color.Bold))
}

// Get share dir name.
func getShareDir() (string, *probe.Error) {
	configDir, err := getMcConfigDir()
	if err != nil {
		return "", err.Trace()
	}

	sharedURLsDataDir := filepath.Join(configDir, globalSharedURLsDataDir)
	return sharedURLsDataDir, nil
}

// Get share dir name or die. (NOTE: This `Die` approach is only OK for mc like tools.).
func mustGetShareDir() string {
	shareDir, err := getShareDir()
	fatalIf(err.Trace(), "Unable to determine share folder.")
	return shareDir
}

// Check if the share dir exists.
func isShareDirExists() bool {
	if _, e := os.Stat(mustGetShareDir()); e != nil {
		return false
	}
	return true
}

// Create config share dir.
func createShareDir() *probe.Error {
	if e := os.MkdirAll(mustGetShareDir(), 0700); e != nil {
		return probe.NewError(e)
	}
	return nil
}

// Get share uploads file.
func getShareUploadsFile() string {
	return filepath.Join(mustGetShareDir(), "uploads.json")
}

// Get share downloads file.
func getShareDownloadsFile() string {
	return filepath.Join(mustGetShareDir(), "downloads.json")
}

// Check if share uploads file exists?.
func isShareUploadsExists() bool {
	if _, e := os.Stat(getShareUploadsFile()); e != nil {
		return false
	}
	return true
}

// Check if share downloads file exists?.
func isShareDownloadsExists() bool {
	if _, e := os.Stat(getShareDownloadsFile()); e != nil {
		return false
	}
	return true
}

// Initialize share uploads file.
func initShareUploadsFile() *probe.Error {
	return newShareDBV1().Save(getShareUploadsFile())
}

// Initialize share downloads file.
func initShareDownloadsFile() *probe.Error {
	return newShareDBV1().Save(getShareDownloadsFile())
}

// Initialize share directory, if not done already.
func initShareConfig() {
	// Share directory.
	if !isShareDirExists() {
		fatalIf(createShareDir().Trace(mustGetShareDir()),
			"Failed to create share `"+mustGetShareDir()+"` folder.")
		if !globalQuiet && !globalJSON {
			console.Infof("Successfully created `%s`.\n", mustGetShareDir())
		}
	}

	// Uploads share file.
	if !isShareUploadsExists() {
		fatalIf(initShareUploadsFile().Trace(getShareUploadsFile()),
			"Failed to initialize share uploads `"+getShareUploadsFile()+"` file.")
		if !globalQuiet && !globalJSON {
			console.Infof("Initialized share uploads `%s` file.\n", getShareUploadsFile())
		}
	}

	// Downloads share file.
	if !isShareDownloadsExists() {
		fatalIf(initShareDownloadsFile().Trace(getShareDownloadsFile()),
			"Failed to initialize share downloads `"+getShareDownloadsFile()+"` file.")
		if !globalQuiet && !globalJSON {
			console.Infof("Initialized share downloads `%s` file.\n", getShareDownloadsFile())
		}
	}
}
