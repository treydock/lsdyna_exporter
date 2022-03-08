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

package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/treydock/lsdyna_exporter/collector"
)

const (
	address = "localhost:19309"
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

func TestMain(m *testing.M) {
	go func() {
		http.Handle("/metrics", metricsHandler(log.NewNopLogger()))
		err := http.ListenAndServe(address, nil)
		if err != nil {
			os.Exit(1)
		}
	}()
	time.Sleep(1 * time.Second)

	exitVal := m.Run()

	os.Exit(exitVal)
}

func TestMetricsHandler(t *testing.T) {
	collector.Lstc_qrun_rExec = func(target string, ctx context.Context) (string, error) {
		return featureStdout, nil
	}
	collector.Lstc_qrun_pExec = func(target string, ctx context.Context) (string, error) {
		return programStdout, nil
	}
	body, err := queryExporter()
	if err != nil {
		t.Fatalf("Unexpected error GET /metrics: %s", err.Error())
	}
	if !strings.Contains(body, "lsdyna_exporter_collect_error{collector=\"feature\"} 0") {
		t.Errorf("Unexpected value for lsdyna_exporter_collect_error")
	}
}

func queryExporter() (string, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/metrics?target=localhost", address))
	if err != nil {
		return "", err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err := resp.Body.Close(); err != nil {
		return "", err
	}
	if want, have := http.StatusOK, resp.StatusCode; want != have {
		return "", fmt.Errorf("want /metrics status code %d, have %d. Body:\n%s", want, have, b)
	}
	return string(b), nil
}
