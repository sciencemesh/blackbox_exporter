// Copyright 2018-2020 CERN
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
//
// In applying this license, CERN does not waive the privileges and immunities
// granted to it by virtue of its status as an Intergovernmental Organization
// or submit itself to any jurisdiction.

package prober

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/sciencemesh/blackbox_exporter/config"
)

const (
	checkOK      = 0
	checkWarning = 1
	checkError   = 2
	checkUnknown = 3
)

type nagiosResult = int

func getProxyEnvVariables(proxy string) []string {
	return []string{
		"HTTPS_PROXY=" + proxy,
		"HTTP_PROXY=" + proxy,
		"https_proxy=" + proxy,
		"http_proxy=" + proxy,
		"USE_PROXY=yes",
		"use_proxy=yes",
	}
}

func resolveNagiosCheckBinary(check string) (string, error) {
	if check == "" {
		return "", fmt.Errorf("no check specified")
	}

	checkBinary := check
	if !filepath.IsAbs(checkBinary) { // Try to resolve the full path to the binary if no absolute one was specified
		// Look in the 'checks' folder of the current directory first; this allows custom checks to be easily deployed alongside the Blackbox Exporter
		curDir, _ := os.Getwd()
		checkBinary = filepath.Join(curDir, "checks", checkBinary)

		if _, err := os.Stat(checkBinary); os.IsNotExist(err) {
			// Try to locate the file anywhere on the system if it doesn't exist in the 'checks' folder
			if checkBinary, err = exec.LookPath(check); err != nil {
				return "", err
			}
		}
	} else {
		if _, err := os.Stat(checkBinary); os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist")
		}
	}

	return checkBinary, nil
}

func obfuscateNagiosCheckArgs(args []string) []string {
	names := []string{"user", "username", "login", "pass", "password", "pwd"}
	var newArgs []string

	for i, arg := range args {
		var delim string
		if idx := strings.IndexAny(arg, " :="); idx != -1 {
			delim = string(arg[idx])
			arg = arg[0:idx]
		}
		argT := strings.TrimLeft(arg, "-")

		argObfuscated := false
		for _, name := range names {
			if strings.EqualFold(argT, name) {
				newArgs = append(newArgs, arg+delim+"*****")
				argObfuscated = true
				break
			}
		}
		if !argObfuscated {
			newArgs = append(newArgs, args[i])
		}
	}

	return newArgs
}

func runNagiosCheck(checkBinary string, ctx context.Context, target string, urlParams url.Values, module config.Module, logger log.Logger) (nagiosResult, string) {
	placeholders := make(map[string]string)
	for key, value := range urlParams {
		if len(value) > 0 {
			placeholders[key] = value[0]
		}
	}
	placeholders["target"] = target
	args := parseNagiosArguments(module.Nagios.Arguments, placeholders)
	level.Debug(logger).Log("msg", "Running Nagios check", "args", obfuscateNagiosCheckArgs(args))
	cmd := exec.CommandContext(ctx, checkBinary, args...) // The context takes care of aborting the process if it is taking too long
	if len(module.Nagios.ProxyURL) > 0 {
		cmd.Env = append(os.Environ(), getProxyEnvVariables(module.Nagios.ProxyURL)...)
	}
	output, _ := cmd.CombinedOutput() // This starts the process

	switch cmd.ProcessState.ExitCode() {
	case -1:
		return checkError, "The process timed out" // Could also mean something else, but this is the most likely reason
	case checkOK, checkWarning, checkError, checkUnknown:
		message, _, _ := splitNagiosOutput(string(output))
		return cmd.ProcessState.ExitCode(), message
	default:
		return checkUnknown, fmt.Sprintf("An unexpected exit code was returned: %v", cmd.ProcessState.ExitCode())
	}
}

func splitNagiosOutput(output string) (message string, perfData []string, log []string) {
	// Nagios output comes line-wise in the form of <message> [ | <performance data> ]
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		tokens := strings.Split(line, "|")
		log = append(log, strings.TrimSpace(tokens[0]))
		for i := 1; i < len(tokens); i += 1 {
			perfData = append(perfData, strings.TrimSpace(tokens[i]))
		}
	}

	if len(log) > 0 {
		// The first line is considered as the check's output message
		message = log[0]
	}

	return
}

func parseNagiosArguments(args []string, values map[string]string) []string {
	// Replace all known placeholders ($...$) in the arguments; unknown placeholders are left as-is
	newArgs := make([]string, 0, len(args))
	for _, arg := range args {
		re := regexp.MustCompile("\\$\\S*\\$")
		newArgs = append(newArgs, re.ReplaceAllStringFunc(arg, func(str string) string {
			if val, ok := values[strings.ToLower(strings.Trim(str, "$"))]; ok {
				return val
			} else {
				return str
			}
		}))
	}

	return newArgs
}

func ProbeNagios(ctx context.Context, target string, values url.Values, module config.Module, registry *prometheus.Registry, logger log.Logger) bool {
	// This Gauge will hold the Nagios result as well as the probe output as a label
	probeNagiosResult := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "probe_nagios_result",
			Help: "Returns the Nagios probe result (0=success, 1=warning, 2=error, 3=unknown)",
		},
		[]string{"output"},
	)
	registry.MustRegister(probeNagiosResult)

	if checkBinary, err := resolveNagiosCheckBinary(module.Nagios.Check); err == nil {
		level.Debug(logger).Log("msg", "Successfully resolved the Nagios check binary", "binary", checkBinary, "check", module.Nagios.Check)

		result, msg := runNagiosCheck(checkBinary, ctx, target, values, module, logger)
		level.Info(logger).Log("msg", "Nagios check finished", "check", module.Nagios.Check, "result", result, "output", msg)
		probeNagiosResult.WithLabelValues(msg).Set(float64(result))

		return result == checkOK || (result == checkWarning && !module.Nagios.TreatWarningsAsFailure)
	} else {
		msg := "Error resolving the Nagios check binary"
		level.Error(logger).Log("msg", msg, "err", err, "check", module.Nagios.Check)
		probeNagiosResult.WithLabelValues(msg).Set(checkUnknown)
		return false
	}
}
