// Copyright (C) 2017-Present CloudFoundry.org Foundation, Inc. All rights reserved.
//
// This program and the accompanying materials are made available under
// the terms of the under the Apache License, Version 2.0 (the "License”);
// you may not use this file except in compliance with the License.
//
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
// License for the specific language governing permissions and limitations
// under the License.

package config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bpm/config"
)

var _ = Describe("Config", func() {
	Describe("ParseJobConfig", func() {
		var configPath string

		BeforeEach(func() {
			configPath = "fixtures/example.yml"
		})

		It("parses a yaml file into a bpm config", func() {
			cfg, err := config.ParseJobConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			expectedMemoryLimit := "100G"
			expectedOpenFilesLimit := uint64(100)

			Expect(cfg.Processes).To(HaveLen(3))

			Expect(cfg.Processes[0].Name).To(Equal("first-process"))
			Expect(cfg.Processes[0].Executable).To(Equal("/var/vcap/packages/program/bin/program-server"))
			Expect(cfg.Processes[0].Args).To(ConsistOf("--port=2424", "--host=\"localhost\""))
			Expect(cfg.Processes[0].Env).To(HaveKeyWithValue("FOO", "BAR"))
			Expect(cfg.Processes[0].Env).To(HaveKeyWithValue("BAZ", "BUZZ"))
			Expect(cfg.Processes[0].Limits.Memory).To(Equal(&expectedMemoryLimit))
			Expect(cfg.Processes[0].Limits.OpenFiles).To(Equal(&expectedOpenFilesLimit))
			Expect(cfg.Processes[0].AdditionalVolumes).To(ConsistOf(
				config.Volume{Path: "/var/vcap/data/program/foobar", Writable: true},
				config.Volume{Path: "/var/vcap/data/alternate-program"},
				config.Volume{Path: "/var/vcap/data/jna-tmp", Writable: true, AllowExecutions: true},
			))
			Expect(cfg.Processes[0].Hooks.PreStart).To(Equal("/var/vcap/jobs/program/bin/pre"))
			Expect(cfg.Processes[0].Capabilities).To(ConsistOf("NET_BIND_SERVICE", "SYS_TIME"))
			Expect(cfg.Processes[0].WorkDir).To(Equal("/I/AM/A/WORKDIR"))
			Expect(cfg.Processes[0].PersistentDisk).To(BeTrue())
			Expect(cfg.Processes[0].EphemeralDisk).To(BeTrue())
			Expect(cfg.Processes[0].Unsafe.Privileged).To(BeTrue())
			Expect(cfg.Processes[0].Unsafe.UnrestrictedVolumes).To(ConsistOf(
				config.Volume{Path: "/", Writable: true},
				config.Volume{Path: "/etc"},
				config.Volume{Path: "/foobar", Writable: true, AllowExecutions: true},
			))

			Expect(cfg.Processes[1].Name).To(Equal("second-process"))
			Expect(cfg.Processes[1].Executable).To(Equal("/I/AM/A/SECOND-EXECUTABLE"))
			Expect(cfg.Processes[1].Hooks).To(BeNil())
			Expect(cfg.Processes[1].Unsafe).To(BeNil())

			Expect(cfg.Processes[2].Name).To(Equal("third-process"))
			Expect(cfg.Processes[2].Executable).To(Equal("/I/AM/A/THIRD-EXECUTABLE"))
			Expect(cfg.Processes[2].Hooks.PreStart).To(BeEmpty())
			Expect(cfg.Processes[2].Unsafe).To(BeNil())
		})

		Context("when reading the file fails", func() {
			BeforeEach(func() {
				configPath = "does-not-exist"
			})

			It("returns an error", func() {
				_, err := config.ParseJobConfig(configPath)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the yaml is invalid", func() {
			BeforeEach(func() {
				configPath = "fixtures/example-invalid-yaml.yml"
			})

			It("returns an error", func() {
				_, err := config.ParseJobConfig(configPath)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Validate", func() {
		var jobCfg *config.JobConfig

		BeforeEach(func() {
			jobCfg = &config.JobConfig{
				Processes: []*config.ProcessConfig{
					{
						Name:              "example",
						Executable:        "executable",
						AdditionalVolumes: []config.Volume{},
					},
				},
			}
		})

		It("does not error on a valid config", func() {
			Expect(jobCfg.Validate([]string{})).To(Succeed())
		})

		Context("when the config has additional_volumes that are not nested in `/var/vcap`", func() {
			It("returns a validation error", func() {
				jobCfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "/var/vcap/data/valid"},
					{Path: "/bin"},
				}
				Expect(jobCfg.Validate([]string{})).To(HaveOccurred())

				jobCfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "/var/vcap/data/valid"},
					{Path: "/var/vcap/invalid"},
				}
				Expect(jobCfg.Validate([]string{})).To(HaveOccurred())

				jobCfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "/var/vcap/data/valid"},
					{Path: "/var/vcap/data"},
				}
				Expect(jobCfg.Validate([]string{})).To(HaveOccurred())

				jobCfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "/var/vcap/store"},
				}
				Expect(jobCfg.Validate([]string{})).To(HaveOccurred())

				jobCfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "//var/vcap/data/valid"},
				}
				Expect(jobCfg.Validate([]string{})).To(HaveOccurred())
			})
		})

		Context("when the config has additional_volumes that conflict with default volumes", func() {
			It("returns a validation error", func() {
				jobCfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "/var/vcap/data/job-name"},
				}
				Expect(jobCfg.Validate([]string{
					"/var/vcap/data/job-name",
				})).To(HaveOccurred())
			})
		})

		Context("when the process does not have a name", func() {
			It("returns an error", func() {
				jobCfg.Processes[0].Name = ""
				Expect(jobCfg.Validate([]string{})).To(HaveOccurred())
			})
		})

		Context("when the config does not have an Executable", func() {
			BeforeEach(func() {
				jobCfg.Processes[0].Executable = ""
			})

			It("returns an error", func() {
				Expect(jobCfg.Validate([]string{})).To(HaveOccurred())
			})
		})
	})
})
