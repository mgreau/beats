// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build mage

package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	devtools "github.com/elastic/beats/dev-tools/mage"
)

func init() {
	devtools.BeatDescription = "Metricbeat is a lightweight shipper for metrics."
}

// Build builds the Beat binary.
func Build() error {
	return devtools.Build(devtools.DefaultBuildArgs())
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	return devtools.GolangCrossBuild(devtools.DefaultGolangCrossBuildArgs())
}

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return devtools.BuildGoDaemon()
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return devtools.CrossBuild()
}

// CrossBuildGoDaemon cross-builds the go-daemon binary using Docker.
func CrossBuildGoDaemon() error {
	return devtools.CrossBuildGoDaemon()
}

// Clean cleans all generated files and build artifacts.
func Clean() error {
	return devtools.Clean()
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	devtools.UseElasticBeatOSSPackaging()
	customizePackaging()

	mg.Deps(Update)
	mg.Deps(CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages(devtools.WithModulesD())
}

// Update updates the generated files (aka make update).
func Update() error {
	return sh.Run("make", "update")
}

// MockedTests runs the HTTP tests using the mocked data inside each {module}/{metricset}/testdata folder.
// Use MODULE={module_name} to run only mocked tests with a single module.
// Use GENERATE=true or GENERATE=1 to regenerate JSON files.
func MockedTests(ctx context.Context) error {
	params := devtools.DefaultGoTestUnitArgs()

	params.ExtraFlags = []string{"github.com/elastic/beats/metricbeat/mb/testing/data/."}

	if module := os.Getenv("MODULE"); module != "" {
		params.ExtraFlags = append(params.ExtraFlags, "-module="+module)
	}

	if generate, _ := strconv.ParseBool(os.Getenv("GENERATE")); generate {
		params.ExtraFlags = append(params.ExtraFlags, "-generate")
	}

	params.Packages = nil

	return devtools.GoTest(ctx, params)
}

// Fields generates a fields.yml for the Beat.
func Fields() error {
	return devtools.GenerateFieldsYAML("module")
}

// GoTestUnit executes the Go unit tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoTestUnit(ctx context.Context) error {
	return devtools.GoTest(ctx, devtools.DefaultGoTestUnitArgs())
}

// GoTestIntegration executes the Go integration tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoTestIntegration(ctx context.Context) error {
	return devtools.GoTest(ctx, devtools.DefaultGoTestIntegrationArgs())
}

// ExportDashboard exports a dashboard and writes it into the correct directory
//
// Required ENV variables:
// * MODULE: Name of the module
// * ID: Dashboard id
func ExportDashboard() error {
	return devtools.ExportDashboard()
}

// FieldsDocs generates docs/fields.asciidoc containing all fields
// (including x-pack).
func FieldsDocs() error {
	inputs := []string{
		devtools.OSSBeatDir("module"),
		devtools.XPackBeatDir("module"),
	}
	output := devtools.CreateDir("build/fields/fields.all.yml")
	if err := devtools.GenerateFieldsYAMLTo(output, inputs...); err != nil {
		return err
	}
	return devtools.Docs.FieldDocs(output)
}

// -----------------------------------------------------------------------------
// Customizations specific to Metricbeat.
// - Include modules.d directory in packages.
// - Disable system/load metricset for Windows.

// customizePackaging modifies the package specs to add the modules.d directory.
// And for Windows it comments out the system/load metricset because it's
// not supported.
func customizePackaging() {
	var (
		archiveModulesDir = "modules.d"
		unixModulesDir    = "/etc/{{.BeatName}}/modules.d"

		modulesDir = devtools.PackageFile{
			Mode:    0644,
			Source:  "modules.d",
			Config:  true,
			Modules: true,
		}
		windowsModulesDir = devtools.PackageFile{
			Mode:    0644,
			Source:  "{{.PackageDir}}/modules.d",
			Config:  true,
			Modules: true,
			Dep: func(spec devtools.PackageSpec) error {
				if err := devtools.Copy("modules.d", spec.MustExpand("{{.PackageDir}}/modules.d")); err != nil {
					return errors.Wrap(err, "failed to copy modules.d dir")
				}

				return devtools.FindReplace(
					spec.MustExpand("{{.PackageDir}}/modules.d/system.yml"),
					regexp.MustCompile(`- load`), `#- load`)
			},
		}
		windowsReferenceConfig = devtools.PackageFile{
			Mode:   0644,
			Source: "{{.PackageDir}}/metricbeat.reference.yml",
			Dep: func(spec devtools.PackageSpec) error {
				err := devtools.Copy("metricbeat.reference.yml",
					spec.MustExpand("{{.PackageDir}}/metricbeat.reference.yml"))
				if err != nil {
					return errors.Wrap(err, "failed to copy reference config")
				}

				return devtools.FindReplace(
					spec.MustExpand("{{.PackageDir}}/metricbeat.reference.yml"),
					regexp.MustCompile(`- load`), `#- load`)
			},
		}
	)

	for _, args := range devtools.Packages {
		switch args.OS {
		case "windows":
			args.Spec.Files[archiveModulesDir] = windowsModulesDir
			args.Spec.ReplaceFile("{{.BeatName}}.reference.yml", windowsReferenceConfig)
		default:
			pkgType := args.Types[0]
			switch pkgType {
			case devtools.TarGz, devtools.Zip, devtools.Docker:
				args.Spec.Files[archiveModulesDir] = modulesDir
			case devtools.Deb, devtools.RPM, devtools.DMG:
				args.Spec.Files[unixModulesDir] = modulesDir
			default:
				panic(errors.Errorf("unhandled package type: %v", pkgType))
			}
		}
	}
}
