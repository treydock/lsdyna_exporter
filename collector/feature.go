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
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	featureTimeout    = kingpin.Flag("collector.feature.timeout", "Timeout for collecting feature information").Default("10").Int()
	Lstc_qrun_rExec   = lstc_qrun_r
	timeNow           = time.Now()
	featureCache      = map[string][]FeatureMetric{}
	featureCacheMutex = sync.RWMutex{}
)

type FeatureMetric struct {
	Name              string
	ExpirationSeconds float64
	Used              float64
	Free              float64
	Total             float64
	Queue             float64
}

type FeatureAggregateMetric struct {
	Licenses float64
	Features int
}

type FeatureCollector struct {
	ExpirationSeconds          *prometheus.Desc
	Used                       *prometheus.Desc
	Free                       *prometheus.Desc
	Total                      *prometheus.Desc
	Queue                      *prometheus.Desc
	AggregateExpirationSeconds *prometheus.Desc
	target                     string
	logger                     log.Logger
}

func NewFeatureExporter(target string, logger log.Logger) Collector {
	return &FeatureCollector{
		ExpirationSeconds: prometheus.NewDesc(prometheus.BuildFQName(namespace, "feature", "expiration_seconds"),
			"Number of seconds till the LTSC licenses expire", []string{"name"}, nil),
		Used: prometheus.NewDesc(prometheus.BuildFQName(namespace, "feature", "used"),
			"Number of used licenses", []string{"name"}, nil),
		Free: prometheus.NewDesc(prometheus.BuildFQName(namespace, "feature", "free"),
			"Number of free licenses", []string{"name"}, nil),
		Total: prometheus.NewDesc(prometheus.BuildFQName(namespace, "feature", "total"),
			"Number of total licenses", []string{"name"}, nil),
		Queue: prometheus.NewDesc(prometheus.BuildFQName(namespace, "feature", "queue"),
			"Number of queued licenses", []string{"name"}, nil),
		AggregateExpirationSeconds: prometheus.NewDesc(prometheus.BuildFQName(namespace, "feature", "aggregate_expiration_seconds"),
			"Aggregate number of seconds for licenses to expire", []string{"licenses", "features"}, nil),
		target: target,
		logger: logger,
	}
}

func (c *FeatureCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.ExpirationSeconds
	ch <- c.Used
	ch <- c.Free
	ch <- c.Total
	ch <- c.Queue
	ch <- c.AggregateExpirationSeconds
}

func (c *FeatureCollector) Collect(ch chan<- prometheus.Metric) {
	level.Debug(c.logger).Log("msg", "Collecting feature metrics")
	collectTime := time.Now()
	timeNow = time.Now()
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
	aggrMap := make(map[float64]*FeatureAggregateMetric)
	for _, m := range metrics {
		ch <- prometheus.MustNewConstMetric(c.ExpirationSeconds, prometheus.GaugeValue, m.ExpirationSeconds, m.Name)
		ch <- prometheus.MustNewConstMetric(c.Used, prometheus.GaugeValue, m.Used, m.Name)
		ch <- prometheus.MustNewConstMetric(c.Free, prometheus.GaugeValue, m.Free, m.Name)
		ch <- prometheus.MustNewConstMetric(c.Total, prometheus.GaugeValue, m.Total, m.Name)
		ch <- prometheus.MustNewConstMetric(c.Queue, prometheus.GaugeValue, m.Queue, m.Name)
		if val, ok := aggrMap[m.ExpirationSeconds]; ok {
			val.Licenses += m.Total
			val.Features++
		} else {
			aggrMap[m.ExpirationSeconds] = &FeatureAggregateMetric{
				Licenses: m.Total,
				Features: 1,
			}
		}
	}
	aggrKeys := make([]float64, 0, len(aggrMap))
	for exp := range aggrMap {
		aggrKeys = append(aggrKeys, exp)
	}
	sort.Float64s(aggrKeys)
	for _, exp := range aggrKeys {
		val := aggrMap[exp]
		ch <- prometheus.MustNewConstMetric(c.AggregateExpirationSeconds, prometheus.GaugeValue,
			exp, fmt.Sprintf("%d", int64(val.Licenses)), strconv.Itoa(val.Features))
	}
	ch <- prometheus.MustNewConstMetric(collectError, prometheus.GaugeValue, float64(errorMetric), "feature")
	ch <- prometheus.MustNewConstMetric(collecTimeout, prometheus.GaugeValue, float64(timeout), "feature")
	ch <- prometheus.MustNewConstMetric(collectDuration, prometheus.GaugeValue, time.Since(collectTime).Seconds(), "feature")
}

func (c *FeatureCollector) collect() ([]FeatureMetric, error) {
	var err error
	var out string
	var metrics []FeatureMetric
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*featureTimeout)*time.Second)
	defer cancel()
	out, err = Lstc_qrun_rExec(c.target, ctx)
	if ctx.Err() == context.DeadlineExceeded {
		if *exporterUseCache {
			metrics = featureReadCache(c.target)
		}
		return metrics, ctx.Err()
	}
	if err != nil {
		if *exporterUseCache {
			metrics = featureReadCache(c.target)
		}
		return metrics, err
	}
	metrics, err = lstc_qrun_r_parse(out)
	if err != nil {
		if *exporterUseCache {
			metrics = featureReadCache(c.target)
		}
		return metrics, err
	}
	if *exporterUseCache {
		featureWriteCache(c.target, metrics)
	}
	return metrics, nil
}

func lstc_qrun_r(target string, ctx context.Context) (string, error) {
	cmd := execCommand(ctx, *lstc_qrun, "-r", "-s", target)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func lstc_qrun_r_parse(out string) ([]FeatureMetric, error) {
	var metrics []FeatureMetric
	lines := strings.Split(out, "\n")
	re := regexp.MustCompile(`^([\w\-]+)\s+(\d{2}/\d{2}/\d{4})\s+(\d+)\s+(\d+)\s+(\d+)\s+\|\s+(\d+).*`)
	for _, l := range lines {
		match := re.FindStringSubmatch(l)
		if len(match) != 7 {
			continue
		}
		var metric FeatureMetric
		metric.Name = match[1]
		expiration, _ := time.Parse("01/02/2006", match[2])
		remainingTime := expiration.Sub(timeNow)
		metric.ExpirationSeconds = remainingTime.Seconds()
		metric.Used, _ = strconv.ParseFloat(match[3], 64)
		metric.Free, _ = strconv.ParseFloat(match[4], 64)
		metric.Total, _ = strconv.ParseFloat(match[5], 64)
		metric.Queue, _ = strconv.ParseFloat(match[6], 64)
		metrics = append(metrics, metric)
	}
	return metrics, nil
}

func featureReadCache(target string) []FeatureMetric {
	var metrics []FeatureMetric
	featureCacheMutex.RLock()
	if cache, ok := featureCache[target]; ok {
		metrics = cache
	}
	featureCacheMutex.RUnlock()
	return metrics
}

func featureWriteCache(target string, metrics []FeatureMetric) {
	featureCacheMutex.Lock()
	featureCache[target] = metrics
	featureCacheMutex.Unlock()
}
