// Copyright 2023 BINARY Members
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/B1NARY-GR0UP/nwa/internal"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/spf13/cobra"
)

const (
	Name    = "nwa"
	Version = "v0.7.1"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   Name,
	Short: "A Simple Yet Powerful Tool for License Header Management",
	Long: `
███╗   ██╗██╗    ██╗ █████╗ 
████╗  ██║██║    ██║██╔══██╗
██╔██╗ ██║██║ █╗ ██║███████║
██║╚██╗██║██║███╗██║██╔══██║
██║ ╚████║╚███╔███╔╝██║  ██║
╚═╝  ╚═══╝ ╚══╝╚══╝ ╚═╝  ╚═╝
`,
	Version: Version,
}

const _levelMute = 12

// Execute executes the root command
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

const (
	_common = "common"
	_config = "config"
)

func init() {
	rootCmd.SetVersionTemplate("{{ .Version }}")
	rootCmd.AddGroup(&cobra.Group{
		ID:    _common,
		Title: "Common Mode Commands:",
	}, &cobra.Group{
		ID:    _config,
		Title: "Config Mode Commands:",
	})
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

const (
	_live   = "live"
	_static = "static"
	_raw    = "raw"
)

type CommonFlags struct {
	Mute     bool
	Verbose  bool
	Fuzzy    bool
	Holder   string
	Year     string
	License  string
	TmplType string
	Tmpl     string // template file path
	Skip     []string
	SPDXIDs  string
}

var defaultCommonFlags = CommonFlags{
	Mute:     false,
	Verbose:  false,
	Fuzzy:    false,
	Holder:   "<COPYRIGHT HOLDER>",
	Year:     fmt.Sprint(time.Now().Year()),
	License:  "apache",
	TmplType: "",
	Tmpl:     "",
	Skip:     []string{},
	SPDXIDs:  "",
}

func setupCommonCmd(common *cobra.Command) {
	rootCmd.AddCommand(common)

	common.Flags().BoolVarP(&defaultCommonFlags.Mute, "mute", "m", defaultCommonFlags.Mute, "mute mode")
	common.Flags().BoolVarP(&defaultCommonFlags.Verbose, "verbose", "V", defaultCommonFlags.Verbose, "verbose mode")
	common.Flags().BoolVarP(&defaultCommonFlags.Fuzzy, "fuzzy", "f", defaultCommonFlags.Fuzzy, "fuzzy matching")
	common.Flags().StringVarP(&defaultCommonFlags.Holder, "copyright", "c", defaultCommonFlags.Holder, "copyright holder")
	common.Flags().StringVarP(&defaultCommonFlags.Year, "year", "y", defaultCommonFlags.Year, "copyright year")
	common.Flags().StringVarP(&defaultCommonFlags.License, "license", "l", defaultCommonFlags.License, "license type")
	common.Flags().StringVarP(&defaultCommonFlags.TmplType, "tmpltype", "T", defaultCommonFlags.TmplType, "template type (live, static, raw)")
	common.Flags().StringVarP(&defaultCommonFlags.Tmpl, "tmpl", "t", defaultCommonFlags.Tmpl, "template file path")
	common.Flags().StringSliceVarP(&defaultCommonFlags.Skip, "skip", "s", defaultCommonFlags.Skip, "skip file path")
	common.Flags().StringVarP(&defaultCommonFlags.SPDXIDs, "spdxids", "i", defaultCommonFlags.SPDXIDs, "spdx ids")

	common.MarkFlagsMutuallyExclusive("mute", "verbose")

	// tmpltype
	common.MarkFlagsRequiredTogether("tmpl", "tmpltype")

	// tmpl
	common.MarkFlagsMutuallyExclusive("license", "tmpl")

	// SPDX IDs
	common.MarkFlagsMutuallyExclusive("license", "spdxids")
}

func setupConfigCmd(config *cobra.Command) {
	rootCmd.AddCommand(config)

	config.Flags().StringVarP(&defaultConfigFlags.Command, "command", "c", defaultConfigFlags.Command, "command to execute")
}

func executeCommonCmd(_ *cobra.Command, args []string, flags CommonFlags, operation internal.Operation) {
	slog.SetLogLoggerLevel(slog.LevelWarn)
	if flags.Verbose {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}
	if flags.Mute {
		slog.SetLogLoggerLevel(_levelMute)
	}

	// validate skip pattern
	for _, s := range flags.Skip {
		if !doublestar.ValidatePattern(s) {
			cobra.CheckErr(fmt.Errorf("--skip (-s) pattern %v is not valid", s))
		}
	}
	// validate path pattern
	for _, arg := range args {
		if !doublestar.ValidatePattern(arg) {
			cobra.CheckErr(fmt.Errorf("path pattern %v is not valid", arg))
		}
	}

	if flags.Tmpl == "" {
		tmpl, err := internal.MatchTmpl(flags.License, flags.SPDXIDs != "")
		if err != nil {
			cobra.CheckErr(err)
		}

		tmplData := &internal.TmplData{
			Holder:  flags.Holder,
			Year:    flags.Year,
			SPDXIDs: flags.SPDXIDs,
		}

		renderedTmpl, err := tmplData.RenderTmpl(tmpl)
		if err != nil {
			cobra.CheckErr(err)
		}

		internal.PrepareTasks(args, renderedTmpl, operation, flags.Skip, false, flags.Fuzzy)
	} else {
		// use customize template
		content, err := os.ReadFile(flags.Tmpl)
		if err != nil {
			cobra.CheckErr(err)
		}

		switch flags.TmplType {
		case _live:
			tmplData := &internal.TmplData{
				Holder:  flags.Holder,
				Year:    flags.Year,
				SPDXIDs: flags.SPDXIDs,
			}

			renderedTmpl, err := tmplData.RenderTmpl(string(content))
			if err != nil {
				cobra.CheckErr(err)
			}

			internal.PrepareTasks(args, renderedTmpl, operation, flags.Skip, false, flags.Fuzzy)
		case _static:
			internal.PrepareTasks(args, content, operation, flags.Skip, false, flags.Fuzzy)
		case _raw:
			internal.PrepareTasks(args, content, operation, flags.Skip, true, flags.Fuzzy)
		default:
			cobra.CheckErr(fmt.Errorf("invalid template type: %v", flags.TmplType))
		}
	}

	internal.ExecuteTasks(operation, flags.Mute)
}
