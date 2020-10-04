package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v5"
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
		Convey("Given a new instance of UsageDB", func() {
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

				start := time.Unix(int64(info["last_update"].(uint)), 0).UTC()
				So(start.Unix(), ShouldEqual, now().Unix())

				fmt.Println("start", start)

				type data struct {
					t        int
					cpuUsage float64
					memUsage float64
				}
				type input struct {
					fetcher  func(u *UsageDb, startTime time.Time, endTime time.Time) (*UsageHistory, error)
					duration time.Duration
					data     func() []data
				}
				tests := []struct {
					name  string
					input input
					want  *UsageHistory
				}{
					{
						name: "hourly test case - nominal",
						input: input{
							fetcher:  (*UsageDb).FetchUsageHourly,
							duration: time.Duration(15) * time.Minute,
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
					{
						name: "monthly test case - nominal",
						input: input{
							fetcher:  (*UsageDb).FetchUsageMonthly,
							duration: time.Duration(2664000) * time.Second * 2, // 2 months
							data: func() []data {
								gofakeit.Seed(1)

								start := 5    // minutes
								step := 5     // minutes
								end := 131400 // minutes ~ 3 months

								var res []data
								for t := start; t < end; t += step {
									res = append(res, data{
										t:        t,
										cpuUsage: gofakeit.Float64Range(0, 100),
										memUsage: gofakeit.Float64Range(0, 100)})
								}

								return res
							},
						},
						want: &UsageHistory{
							CPUUsage: []*UsageHistoryItem{
								{
									DateUTC: time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC),
									Value:   3676.8308439867487,
								},
								{
									DateUTC: time.Date(start.Year(), start.Month()+1, 1, 0, 0, 0, 0, time.UTC),
									Value:   36981.556600389304,
								},
								{
									DateUTC: time.Date(start.Year(), start.Month()+2, 1, 0, 0, 0, 0, time.UTC),
									Value:   32673.758350134198,
								},
							},
							MEMUsage: []*UsageHistoryItem{
								{
									DateUTC: time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC),
									Value:   3615.1705261647344,
								},
								{
									DateUTC: time.Date(start.Year(), start.Month()+1, 1, 0, 0, 0, 0, time.UTC),
									Value:   37462.62516932022,
								},
								{
									DateUTC: time.Date(start.Year(), start.Month()+2, 1, 0, 0, 0, 0, time.UTC),
									Value:   32836.23815938188,
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
								_ = usageDb.UpdateRRD(start.Add(time.Duration(datum.t)*time.Minute), datum.cpuUsage, datum.memUsage)

								// So(err, ShouldBeNil)
							}

							end := start.Add(test.input.duration)
							Convey(fmt.Sprintf("When fetching usage for interval %s - %s (%s)", start, end, test.input.duration), func() {
								usage, err := test.input.fetcher(usageDb, start, end)

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
