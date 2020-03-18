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
	"os/exec"

	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "lsdyna"
)

var (
	lstc_qrun        = kingpin.Flag("path.lstc_qrun", "Path to lstc_qrun").Required().String()
	exporterUseCache = kingpin.Flag("exporter.use-cache", "Use cached metrics if commands timeout or produce errors").Default("false").Bool()
	execCommand      = exec.CommandContext
	collectDuration  = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "exporter", "collector_duration_seconds"),
		"Collector time duration.",
		[]string{"collector"}, nil)
	collectError = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "exporter", "collect_error"),
		"Indicates if error has occurred during collection",
		[]string{"collector"}, nil)
	collecTimeout = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "exporter", "collect_timeout"),
		"Indicates the collector timed out",
		[]string{"collector"}, nil)
)

type Collector interface {
	// Get new metrics and expose them via prometheus registry.
	Describe(ch chan<- *prometheus.Desc)
	Collect(ch chan<- prometheus.Metric)
}
