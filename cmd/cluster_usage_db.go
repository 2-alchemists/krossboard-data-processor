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

// UsageDb holds a wrapper on a RRD database file along with appropriated settinfgs to store a usage data
type UsageDb struct {
	RRDFile  string
	Step     uint
	MinValue float64
	MaxValue float64
	Xfs      float64
}

// UsageHistoryItem holds a resource usage at a timestamp
type UsageHistoryItem struct {
	DateUTC time.Time `json:"dateUTC"`
	Value   float64   `json:"value"`
}

// UsageHistory holds resource usage history for all kinds of managed resources (CPU, memory)
type UsageHistory struct {
	CPUUsage []*UsageHistoryItem `json:"cpuUsage"`
	MEMUsage []*UsageHistoryItem `json:"memUsage"`
}

// now points to the regular time.Now but offers a way to stub out the function inside tests.
var now = time.Now

// NewUsageDb instanciate a new UsageDb object wrapper
func NewUsageDb(dbname string) *UsageDb {
	return &UsageDb{
		RRDFile:  dbname,
		Step:     uint(RRDStorageStep300Secs),
		MinValue: 0,
		MaxValue: 100,
		Xfs:      2 * RRDStorageStep300Secs,
	}
}

// CreateRRD create a new RRD database
func (m *UsageDb) CreateRRD() error {
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
func (m *UsageDb) UpdateRRD(ts time.Time, cpuUsage float64, memUsage float64) error {
	rrdUpdater := rrd.NewUpdater(m.RRDFile)
	return rrdUpdater.Update(ts, cpuUsage, memUsage)
}

// FetchUsageHourly retrieves from the managed RRD file, 5 minutes-step usage data between startTimeUTC and endTimeUTC
func (m *UsageDb) FetchUsage5Minutes(startTimeUTC time.Time, endTimeUTC time.Time) (*UsageHistory, error) {
	return m.FetchUsage(startTimeUTC, endTimeUTC, time.Duration(RRDStorageStep300Secs)*time.Second)
}

// FetchUsageHourly retrieves from the managed RRD file, hour-step usage data between startTimeUTC and endTimeUTC
func (m *UsageDb) FetchUsageHourly(startTimeUTC time.Time, endTimeUTC time.Time) (*UsageHistory, error) {
	const duration25Hours = 25 * time.Hour

	if endTimeUTC.Sub(startTimeUTC) < duration25Hours {
		return m.FetchUsage5Minutes(startTimeUTC, endTimeUTC)
	}

	return m.FetchUsage(startTimeUTC, endTimeUTC, time.Duration(RRDStorageStep3600Secs)*time.Second)
}

// FetchUsageMonthly retrieves from the managed RRD file, month-step usage data between startTimeUTC and endTimeUTC
func (m *UsageDb) FetchUsageMonthly(startTimeUTC time.Time, endTimeUTC time.Time) (*UsageHistory, error) {
	usages, err := m.FetchUsageHourly(startTimeUTC, endTimeUTC)
	if err != nil {
		return nil, err
	}

	computeCumulativeMonth := func(items []*UsageHistoryItem) []*UsageHistoryItem {
		var usages []*UsageHistoryItem

		for _, usage := range items {
			last := len(usages) - 1
			if last > -1 &&
				usages[last].DateUTC.Year() == usage.DateUTC.Year() &&
				usages[last].DateUTC.Month() == usage.DateUTC.Month() {

				v := usages[last]
				v.Value += usage.Value
			} else {
				usages = append(usages, &UsageHistoryItem{
					DateUTC: time.Date(usage.DateUTC.Year(), usage.DateUTC.Month(), 1, 0, 0, 0, 0, time.UTC),
					Value:   usage.Value,
				})
			}
		}

		return usages
	}

	return &UsageHistory{
			computeCumulativeMonth(usages.CPUUsage),
			computeCumulativeMonth(usages.MEMUsage),
		},
		nil
}

// FetchUsage retrieves from the managed RRD file, usage data between startTimeUTC and endTimeUTC with the given step
func (m *UsageDb) FetchUsage(startTimeUTC time.Time, endTimeUTC time.Time, duration time.Duration) (*UsageHistory, error) {
	rrdEndTime := RoundTime(endTimeUTC, duration)
	rrdStartTime := RoundTime(startTimeUTC, duration)
	rrdFetchRes, err := rrd.Fetch(m.RRDFile, "AVERAGE", rrdStartTime, rrdEndTime, duration)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read rrd file")
	}
	defer rrdFetchRes.FreeValues()

	var cpuUsage []*UsageHistoryItem
	var memUsage []*UsageHistoryItem
	rrdRow := 0
	for ti := rrdFetchRes.Start.Add(rrdFetchRes.Step); ti.Before(rrdEndTime) || ti.Equal(rrdEndTime); ti = ti.Add(rrdFetchRes.Step) {
		cpu := rrdFetchRes.ValueAt(0, rrdRow)
		mem := rrdFetchRes.ValueAt(1, rrdRow)
		if !math.IsNaN(cpu) && !math.IsNaN(mem) {
			cpuUsage = append(cpuUsage, &UsageHistoryItem{
				DateUTC: ti,
				Value:   cpu,
			})
			memUsage = append(memUsage, &UsageHistoryItem{
				DateUTC: ti,
				Value:   mem,
			})
		}
		rrdRow++
	}

	return &UsageHistory{cpuUsage, memUsage}, nil
}
