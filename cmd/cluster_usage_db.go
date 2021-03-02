package cmd

import (
	"math"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/ziutek/rrd"
)

const (
	// RRDStorageStep300Secs constant defining a 5-min storage step for RRD databases
	RRDStorageStep300Secs = 300
	// RRDStorageStep3600Secs constant defining a 1-hour storage step for RRD databases
	RRDStorageStep3600Secs = 3600
)

// NamespaceUsageDb holds a wrapper on a RRD database file along with appropriated settinfgs to store a usage data
type NamespaceUsageDb struct {
	RRDFile  string
	Step     uint
	MinValue float64
	MaxValue float64
	Xfs      float64
}

// ResourceUsageItem holds a resource usage at a timestamp
type ResourceUsageItem struct {
	DateUTC time.Time `json:"dateUTC"`
	Value   float64   `json:"value"`
}

// NamespaceUsageHistory holds resource usage history for all kinds of managed resources (CPU, memory)
type NamespaceUsageHistory struct {
	CPUUsage []*ResourceUsageItem `json:"cpuUsage"`
	MEMUsage []*ResourceUsageItem `json:"memUsage"`
}

// NodeUsage holds an instance of node usage as processed by kube-opex-analytics
type NodeUsage struct {
	DateUTC string `json:"dateUTC,omitempty"`
	Name string `json:"name,omitempty"`
	State string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
	CPUCapacity float64 `json:"cpuCapacity,omitempty"`
	CPUAllocatable float64 `json:"cpuAllocatable,omitempty"`
	CPUUsage float64 `json:"cpuUsage,omitempty"`
	MEMCapacity float64 `json:"memCapacity,omitempty"`
	MEMAllocatable float64 `json:"memAllocatable,omitempty"`
	MEMUsage float64 `json:"memUsage,omitempty"`
}

// K8sClusterUsage holds used and non-allocatable memory and CPU resource of a K8s cluster
type K8sClusterUsage struct {
	ClusterName       string  `json:"clusterName"`
	CPUUsed           float64 `json:"cpuUsed"`
	MemUsed           float64 `json:"memUsed"`
	CPUNonAllocatable float64 `json:"cpuNonAllocatable"`
	MemNonAllocatable float64 `json:"memNonAllocatable"`
	OutToDate         bool    `json:"outToDate"`
}

// now points to the regular time.Now but offers a way to stub out the function inside tests.
var now = time.Now

// NewUsageDb instanciate a new NamespaceUsageDb object wrapper
func NewUsageDb(dbname string) *NamespaceUsageDb {
	return &NamespaceUsageDb{
		RRDFile:  dbname,
		Step:     uint(RRDStorageStep300Secs),
		MinValue: 0,
		MaxValue: 100,
		Xfs:      2 * RRDStorageStep300Secs,
	}
}

// CreateRRD create a new RRD database
func (m *NamespaceUsageDb) CreateRRD() error {
	rrdCreator := rrd.NewCreator(m.RRDFile, now(), m.Step)
	rrdCreator.RRA("AVERAGE", 0.5, 1, 4032)               // 14 days - 5-minute resolution
	rrdCreator.RRA("AVERAGE", 0.5, 12 /* 1 hour */, 8880) // 1 year - 1-hour resolution
	rrdCreator.DS("cpu_usage", "GAUGE", m.Step, m.MinValue, m.MaxValue)
	rrdCreator.DS("mem_usage", "GAUGE", m.Step, m.MinValue, m.MaxValue)
	err := rrdCreator.Create(false)
	if os.IsExist(err) {
		return nil
	}
	return err
}

// UpdateRRD adds a new entry into a RRD database
func (m *NamespaceUsageDb) UpdateRRD(ts time.Time, cpuUsage float64, memUsage float64) error {
	rrdUpdater := rrd.NewUpdater(m.RRDFile)
	return rrdUpdater.Update(ts, cpuUsage, memUsage)
}

// FetchUsageHourly retrieves from the managed RRD file, 5 minutes-step usage data between startTimeUTC and endTimeUTC
func (m *NamespaceUsageDb) FetchUsage5Minutes(startTimeUTC time.Time, endTimeUTC time.Time) (*NamespaceUsageHistory, error) {
	return m.FetchUsage(startTimeUTC, endTimeUTC, time.Duration(RRDStorageStep300Secs)*time.Second)
}

// FetchUsageHourly retrieves from the managed RRD file, hour-step usage data between startTimeUTC and endTimeUTC
func (m *NamespaceUsageDb) FetchUsageHourly(startTimeUTC time.Time, endTimeUTC time.Time) (*NamespaceUsageHistory, error) {
	const duration25Hours = 25 * time.Hour

	if endTimeUTC.Sub(startTimeUTC) < duration25Hours {
		return m.FetchUsage5Minutes(startTimeUTC, endTimeUTC)
	}

	return m.FetchUsage(startTimeUTC, endTimeUTC, time.Duration(RRDStorageStep3600Secs)*time.Second)
}

// FetchUsageMonthly retrieves from the managed RRD file, month-step usage data between startTimeUTC and endTimeUTC
func (m *NamespaceUsageDb) FetchUsageMonthly(startTimeUTC time.Time, endTimeUTC time.Time) (*NamespaceUsageHistory, error) {
	usages, err := m.FetchUsageHourly(startTimeUTC, endTimeUTC)
	if err != nil {
		return nil, err
	}

	return &NamespaceUsageHistory{
			computeCumulativeMonth(usages.CPUUsage),
			computeCumulativeMonth(usages.MEMUsage),
		},
		nil
}

// FetchUsage retrieves from the managed RRD file, usage data between startTimeUTC and endTimeUTC with the given step
func (m *NamespaceUsageDb) FetchUsage(startTimeUTC time.Time, endTimeUTC time.Time, duration time.Duration) (*NamespaceUsageHistory, error) {
	rrdEndTime := RoundTime(endTimeUTC, duration)
	rrdStartTime := RoundTime(startTimeUTC, duration)
	rrdFetchRes, err := rrd.Fetch(m.RRDFile, "AVERAGE", rrdStartTime, rrdEndTime, duration)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read rrd file")
	}
	defer rrdFetchRes.FreeValues()

	var cpuUsage []*ResourceUsageItem
	var memUsage []*ResourceUsageItem
	rrdRow := 0
	for ti := rrdFetchRes.Start.Add(rrdFetchRes.Step); ti.Before(rrdEndTime) || ti.Equal(rrdEndTime); ti = ti.Add(rrdFetchRes.Step) {
		cpu := rrdFetchRes.ValueAt(0, rrdRow)
		mem := rrdFetchRes.ValueAt(1, rrdRow)
		if !math.IsNaN(cpu) && !math.IsNaN(mem) {
			cpuUsage = append(cpuUsage, &ResourceUsageItem{
				DateUTC: ti,
				Value:   cpu,
			})
			memUsage = append(memUsage, &ResourceUsageItem{
				DateUTC: ti,
				Value:   mem,
			})
		}
		rrdRow++
	}

	return &NamespaceUsageHistory{cpuUsage, memUsage}, nil
}

// computeCumulativeMonth compute the cumulative data per month.
func computeCumulativeMonth(items []*ResourceUsageItem) []*ResourceUsageItem {
	usages := []*ResourceUsageItem{}

	for _, usage := range items {
		last := len(usages) - 1
		if last > -1 &&
			usages[last].DateUTC.Year() == usage.DateUTC.Year() &&
			usages[last].DateUTC.Month() == usage.DateUTC.Month() {

			v := usages[last]
			v.Value += usage.Value
		} else {
			usages = append(usages, &ResourceUsageItem{
				DateUTC: time.Date(usage.DateUTC.Year(), usage.DateUTC.Month(), 1, 0, 0, 0, 0, time.UTC),
				Value:   usage.Value,
			})
		}
	}

	return usages
}