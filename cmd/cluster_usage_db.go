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
	now := time.Now()
	rrdCreator := rrd.NewCreator(m.RRDFile, now, m.Step)
	rrdCreator.RRA("AVERAGE", 0.5, 1, 4032)
	rrdCreator.RRA("AVERAGE", 0.5, 12, 8880)
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

// FetchUsage retrieves from the managed RRD file, usage data between startTimeUTC and endTimeUTC
func (m *UsageDb) FetchUsage(startTimeUTC time.Time, endTimeUTC time.Time) (*UsageHistory, error) {
	const duration25Hours = 25 * time.Hour
	rrdFetchStep := int64(RRDStorageStep3600Secs)
	if endTimeUTC.Sub(startTimeUTC) < time.Duration(duration25Hours) {
		rrdFetchStep = int64(RRDStorageStep300Secs)
	}
	rrdEndTime := time.Unix(int64(int64(endTimeUTC.Unix()/rrdFetchStep)*rrdFetchStep), 0)
	rrdStartTime := time.Unix(int64(int64(startTimeUTC.Unix()/rrdFetchStep)*rrdFetchStep), 0)
	rrdFetchRes, err := rrd.Fetch(m.RRDFile, "AVERAGE", rrdStartTime, rrdEndTime, time.Duration(rrdFetchStep)*time.Second)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read rrd file")
	}
	defer rrdFetchRes.FreeValues()

	cpuUsage := []*UsageHistoryItem{}
	memUsage := []*UsageHistoryItem{}
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
