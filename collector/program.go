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
	"bytes"
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	programTimeout  = kingpin.Flag("collector.programs.timeout", "Timeout for collecting programs information").Default("10").Int()
	Lstc_qrun_pExec = lstc_qrun_p
)

type ProgramMetric struct {
	User    string
	Program string
	Used    float64
}

type ProgramCollector struct {
	UserUsed *prometheus.Desc
	target   string
	logger   log.Logger
}

func NewProgramExporter(target string, logger log.Logger) Collector {
	return &ProgramCollector{
		UserUsed: prometheus.NewDesc(prometheus.BuildFQName(namespace, "feature", "user_used"),
			"Number of licenses used by a user for a given feature", []string{"feature", "user"}, nil),
		target: target,
		logger: logger,
	}
}

func (c *ProgramCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.UserUsed
}

func (c *ProgramCollector) Collect(ch chan<- prometheus.Metric) {
	level.Debug(c.logger).Log("msg", "Collecting programs metrics")
	collectTime := time.Now()
	timeout := 0
	errorMetric := 0
	metrics, err := c.collect()
	if err == context.DeadlineExceeded {
		level.Error(c.logger).Log("msg", "Timeout executing lstc_qrun")
		timeout = 1
	} else if err != nil {
		level.Error(c.logger).Log("msg", err)
		errorMetric = 1
	}

	userUsed := make(map[string]map[string]float64)
	for _, m := range metrics {
		if userUsed[m.Program] == nil {
			userUsed[m.Program] = map[string]float64{}
		}
		userUsed[m.Program][m.User] += m.Used
	}
	for program, usermap := range userUsed {
		for user, used := range usermap {
			ch <- prometheus.MustNewConstMetric(c.UserUsed, prometheus.GaugeValue, used, program, user)
		}
	}

	ch <- prometheus.MustNewConstMetric(collectError, prometheus.GaugeValue, float64(errorMetric), "program")
	ch <- prometheus.MustNewConstMetric(collecTimeout, prometheus.GaugeValue, float64(timeout), "program")
	ch <- prometheus.MustNewConstMetric(collectDuration, prometheus.GaugeValue, time.Since(collectTime).Seconds(), "program")
}

func (c *ProgramCollector) collect() ([]ProgramMetric, error) {
	var err error
	var out string
	var metrics []ProgramMetric
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*programTimeout)*time.Second)
	defer cancel()
	out, err = Lstc_qrun_pExec(c.target, ctx)
	if ctx.Err() == context.DeadlineExceeded {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	metrics, err = lstc_qrun_p_parse(out, c.logger)
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

func lstc_qrun_p(target string, ctx context.Context) (string, error) {
	cmd := execCommand(ctx, *lstc_qrun, "-s", target, "-p")
	var out bytes.Buffer
	cmd.Stdout = &out
	// Non-errors have non-zero exit status, so ignore errors
	_ = cmd.Run()
	output := out.String()
	re := regexp.MustCompile(`.*ERROR (.*)`)
	match := re.FindStringSubmatch(output)
	if len(match) == 2 {
		return "", errors.New(match[1])
	}
	return output, nil
}

func lstc_qrun_p_parse(out string, logger log.Logger) ([]ProgramMetric, error) {
	var metrics []ProgramMetric
	var err error
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		items := strings.Fields(l)
		if len(items) != 8 {
			continue
		}
		var metric ProgramMetric
		metric.User = items[0]
		metric.Program = items[2]
		metric.Used, err = strconv.ParseFloat(items[7], 64)
		if err != nil {
			level.Error(logger).Log("msg", "error converting to float", "line", l, "item", items[7])
			continue
		}
		metrics = append(metrics, metric)
	}
	return metrics, nil
}
