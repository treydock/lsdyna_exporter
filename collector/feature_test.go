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
	"strings"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	featureStdout = `
Using user specified server 31011@haswell2

LICENSE INFORMATION

PROGRAM          EXPIRATION CPUS  USED   FREE    MAX | QUEUE
---------------- ----------      ----- ------ ------ | -----
LS-DYNA          07/31/2020          0   2000   2000 |     0
MPPDYNA          07/31/2020          0   2000   2000 |     0
                   LICENSE GROUP     0   2000   2000 |     0

`
)

func TestFeatureParse(t *testing.T) {
	mockNow, _ := time.Parse("01/02/2006", "07/01/2020")
	timeNow = mockNow
	metrics, err := lstc_qrun_r_parse(featureStdout)
	if err != nil {
		t.Errorf("Unexpected err: %s", err.Error())
		return
	}
	if len(metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(metrics))
		return
	}
	if val := metrics[0].Name; val != "LS-DYNA" {
		t.Errorf("Unexpected name %s", val)
	}
	if val := metrics[0].ExpirationSeconds; val != 2592000 {
		t.Errorf("Unexpected expiration seconds %v", val)
	}
	if val := metrics[0].Used; val != 0 {
		t.Errorf("Unexpected used %v", val)
	}
	if val := metrics[0].Free; val != 2000 {
		t.Errorf("Unexpected free %v", val)
	}
	if val := metrics[0].Total; val != 2000 {
		t.Errorf("Unexpected total %v", val)
	}
	if val := metrics[0].Queue; val != 0 {
		t.Errorf("Unexpected queue %v", val)
	}
}

func TestFeatureCollector(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--path.lstc_qrun=/dne"}); err != nil {
		t.Fatal(err)
	}
	mockNow, _ := time.Parse("01/02/2006", "07/01/2020")
	timeNow = mockNow
	Lstc_qrun_rExec = func(target string, ctx context.Context) (string, error) {
		return featureStdout, nil
	}
	expected := `
    # HELP lsdyna_exporter_collect_error Indicates if error has occurred during collection
    # TYPE lsdyna_exporter_collect_error gauge
    lsdyna_exporter_collect_error{collector="feature"} 0
    # HELP lsdyna_exporter_collect_timeout Indicates the collector timed out
    # TYPE lsdyna_exporter_collect_timeout gauge
    lsdyna_exporter_collect_timeout{collector="feature"} 0
	# HELP lsdyna_feature_free Number of free licenses
	# TYPE lsdyna_feature_free gauge
	lsdyna_feature_free{index="0",name="LS-DYNA"} 2000
	lsdyna_feature_free{index="1",name="MPPDYNA"} 2000
	# HELP lsdyna_feature_queue Number of queued licenses
	# TYPE lsdyna_feature_queue gauge
	lsdyna_feature_queue{index="0",name="LS-DYNA"} 0
	lsdyna_feature_queue{index="1",name="MPPDYNA"} 0
	# HELP lsdyna_feature_total Number of total licenses
	# TYPE lsdyna_feature_total gauge
	lsdyna_feature_total{index="0",name="LS-DYNA"} 2000
	lsdyna_feature_total{index="1",name="MPPDYNA"} 2000
	# HELP lsdyna_feature_used Number of used licenses
	# TYPE lsdyna_feature_used gauge
	lsdyna_feature_used{index="0",name="LS-DYNA"} 0
	lsdyna_feature_used{index="1",name="MPPDYNA"} 0
	`
	collector := NewFeatureExporter("localhost", log.NewNopLogger())
	gatherers := setupGatherer(collector)
	if val := testutil.CollectAndCount(collector); val != 14 {
		t.Errorf("Unexpected collection count %d, expected 14", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"lsdyna_feature_free", "lsdyna_feature_queue", "lsdyna_feature_total", "lsdyna_feature_used",
		"lsdyna_exporter_collect_error", "lsdyna_exporter_collect_timeout"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}
