package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
	"github.com/ziutek/rrd"
)

func TestSpec(t *testing.T) {
	Convey("Given valid base config settings", t, func() {
		viper.Set("krossboard_root_dir", fmt.Sprintf("%s/.krossboard", UserHomeDir()))
		viper.Set("krossboard_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("krossboard_root_dir")))

		Convey("Empty tests", func() {
			So(nil, ShouldBeNil)
		})
	})
}

func TestUsageDb(t *testing.T) {
	Convey("Given a temporary file", t, func() {
		tempDir, err := ioutil.TempDir("", "tests")

		So(tempDir, ShouldNotBeNil)
		So(err, ShouldBeNil)

		dbName := path.Join(tempDir, "test.db")
		Convey(fmt.Sprintf("Given a new instance of UsageDB with dbName: '%s'", dbName), func() {
			usageDb := NewUsageDb(dbName)

			So(usageDb, ShouldNotBeNil)

			Convey("Given a new instance of RRD", func() {
				now = func() time.Time {
					return time.Unix(1601233498, 0)
				}

				err := usageDb.CreateRRD()
				So(err, ShouldBeNil)

				info, err := rrd.Info(usageDb.RRDFile)
				So(err, ShouldBeNil)

				start := time.Unix(int64(info["last_update"].(uint)), 0)
				So(start.Unix(), ShouldEqual, now().Unix())

				type data struct {
					t        int
					cpuUsage float64
					memUsage float64
				}
				type input struct {
					fetcher func(u *UsageDb, startTimeUTC time.Time, endTimeUTC time.Time) (*UsageHistory, error)
					period  time.Duration
					data    func() []data
				}
				tests := []struct {
					name  string
					input input
					want  *UsageHistory
				}{
					{
						name: "hourly test case - nominal",
						input: input{
							fetcher: (*UsageDb).FetchUsageHourly,
							period:  time.Duration(15) * time.Minute,
							data: func() []data {
								return []data{
									{t: 5, cpuUsage: 10, memUsage: 15},
									{t: 10, cpuUsage: 20, memUsage: 25},
									{t: 15, cpuUsage: 30, memUsage: 35},
								}
							},
						},
						want: &UsageHistory{
							CPUUsage: []*UsageHistoryItem{
								{
									DateUTC: RoundTime(start.Add(time.Duration(5*2)*time.Minute), time.Duration(usageDb.Step)*time.Second),
									Value:   10.066666666666666,
								},
								{
									DateUTC: RoundTime(start.Add(time.Duration(5*3)*time.Minute), time.Duration(usageDb.Step)*time.Second),
									Value:   20.066666666666666,
								},
							},
							MEMUsage: []*UsageHistoryItem{
								{
									DateUTC: RoundTime(start.Add(time.Duration(5*2)*time.Minute), time.Duration(usageDb.Step)*time.Second),
									Value:   15.066666666666666,
								},
								{
									DateUTC: RoundTime(start.Add(time.Duration(5*3)*time.Minute), time.Duration(usageDb.Step)*time.Second),
									Value:   25.066666666666666,
								},
							},
						},
					},
				}

				for _, test := range tests {
					Convey(fmt.Sprintf("Given the test case '%s'", test.name), func() {
						Convey("Given some values added in the instance of RRD", func() {
							data := test.input.data()
							for _, datum := range data {
								err := usageDb.UpdateRRD(start.Add(time.Duration(datum.t)*time.Minute), datum.cpuUsage, datum.memUsage)

								So(err, ShouldBeNil)
							}

							end := start.Add(test.input.period)
							Convey(fmt.Sprintf("When fetching usage for interval %s - %s (%s)", start, end, test.input.period), func() {
								usage, err := usageDb.FetchUsageHourly(start, end)

								So(err, ShouldBeNil)

								Convey("Then average values retrieved are the ones expected", func() {
									So(usage, ShouldResemble, test.want)
								})
							})
						})
					})
				}
			})
		})

		Reset(func() {
			_ = os.RemoveAll(dbName)
		})
	})
}

// func TestSpec(t *testing.T) {
// 	Convey("Given valid base config settings", t, func() {
// 		viper.Set("krossboard_root_dir", fmt.Sprintf("%s/.kube-opex-analytics-mc", UserHomeDir()))
// 		viper.Set("krossboard_root_data_dir", fmt.Sprintf("%s/data", viper.GetString("krossboard_root_dir")))

// 		Convey("If a given cluster name is valid", func() {
// 			clusterName := "toto"

// 			Convey("The call getAllClustersCurrentUsage  should succeed", func() {
// 				_, err := getAllClustersCurrentUsage()
// 				So(err, ShouldBeNil)
// 			})
// 		})
// 	})
// }
