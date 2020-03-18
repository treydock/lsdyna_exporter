// Copyright 2020 Trey Dockendorf
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	programStdout = `
Using user specified server 31011@haswell2


                     Running Programs

    User             Host          Program              Started       # procs
-----------------------------------------------------------------------------
     hna    84212@o0284.ten.osc.ed MPPDYNA          Tue Mar 17 16:18    28
sciappst    85606@o0579.ten.osc.ed MPPDYNA          Tue Mar 17 16:22    10
No programs queued

`
)

func TestProgramParse(t *testing.T) {
	metrics, err := lstc_qrun_parse(programStdout, log.NewNopLogger())
	if err != nil {
		t.Errorf("Unexpected err: %s", err.Error())
		return
	}
	if len(metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(metrics))
		return
	}
	if val := metrics[0].User; val != "hna" {
		t.Errorf("Unexpected name %v", val)
	}
	if val := metrics[0].Program; val != "MPPDYNA" {
		t.Errorf("Unexpected program %v", val)
	}
	if val := metrics[0].Used; val != 28 {
		t.Errorf("Unexpected used %v", val)
	}
}

func TestProgramCollector(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--path.lstc_qrun=/dne"}); err != nil {
		t.Fatal(err)
	}
	Lstc_qrunExec = func(target string, ctx context.Context) (string, error) {
		return programStdout, nil
	}
	expected := `
    # HELP lsdyna_exporter_collect_error Indicates if error has occurred during collection
    # TYPE lsdyna_exporter_collect_error gauge
    lsdyna_exporter_collect_error{collector="program"} 0
    # HELP lsdyna_exporter_collect_timeout Indicates the collector timed out
    # TYPE lsdyna_exporter_collect_timeout gauge
    lsdyna_exporter_collect_timeout{collector="program"} 0
	# HELP lsdyna_feature_user_used Number of licenses used by a user for a given feature
	# TYPE lsdyna_feature_user_used gauge
	lsdyna_feature_user_used{feature="MPPDYNA", user="hna"} 28
	lsdyna_feature_user_used{feature="MPPDYNA", user="sciappst"} 10
	`
	collector := NewProgramExporter("localhost", log.NewNopLogger())
	gatherers := setupGatherer(collector)
	if val := testutil.CollectAndCount(collector); val != 5 {
		t.Errorf("Unexpected collection count %d, expected 5", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"lsdyna_feature_user_used", "lsdyna_exporter_collect_error", "lsdyna_exporter_collect_timeout"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestProgramCollectorError(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--path.lstc_qrun=/dne"}); err != nil {
		t.Fatal(err)
	}
	Lstc_qrunExec = func(target string, ctx context.Context) (string, error) {
		return "", fmt.Errorf("Error")
	}
	expected := `
    # HELP lsdyna_exporter_collect_error Indicates if error has occurred during collection
    # TYPE lsdyna_exporter_collect_error gauge
    lsdyna_exporter_collect_error{collector="program"} 1
    # HELP lsdyna_exporter_collect_timeout Indicates the collector timed out
    # TYPE lsdyna_exporter_collect_timeout gauge
    lsdyna_exporter_collect_timeout{collector="program"} 0
	`
	collector := NewProgramExporter("localhost", log.NewNopLogger())
	gatherers := setupGatherer(collector)
	if val := testutil.CollectAndCount(collector); val != 3 {
		t.Errorf("Unexpected collection count %d, expected 3", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"lsdyna_feature_user_used", "lsdyna_exporter_collect_error", "lsdyna_exporter_collect_timeout"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestProgramCollectorTimeout(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--path.lstc_qrun=/dne"}); err != nil {
		t.Fatal(err)
	}
	Lstc_qrunExec = func(target string, ctx context.Context) (string, error) {
		return "", context.DeadlineExceeded
	}
	expected := `
    # HELP lsdyna_exporter_collect_error Indicates if error has occurred during collection
    # TYPE lsdyna_exporter_collect_error gauge
    lsdyna_exporter_collect_error{collector="program"} 0
    # HELP lsdyna_exporter_collect_timeout Indicates the collector timed out
    # TYPE lsdyna_exporter_collect_timeout gauge
    lsdyna_exporter_collect_timeout{collector="program"} 1
	`
	collector := NewProgramExporter("localhost", log.NewNopLogger())
	gatherers := setupGatherer(collector)
	if val := testutil.CollectAndCount(collector); val != 3 {
		t.Errorf("Unexpected collection count %d, expected 3", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"lsdyna_feature_user_used", "lsdyna_exporter_collect_error", "lsdyna_exporter_collect_timeout"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func Test_lstc_qrun_exec(t *testing.T) {
	execCommand = fakeExecCommand
	mockedExitStatus = 0
	mockedStdout = "foo"
	defer func() { execCommand = exec.CommandContext }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := lstc_qrun_exec("host", ctx)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if out != mockedStdout {
		t.Errorf("Unexpected out: %s", out)
	}
}
