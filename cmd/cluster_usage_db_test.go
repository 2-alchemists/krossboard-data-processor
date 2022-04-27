/*
Copyright (c) 2020 2Alchemists SAS.

This file is part of Krossboard.

Krossboard is free software: you can redistribute it and/or modify it under the terms of the
GNU General Public License as published by the Free Software Foundation, either version 3
of the License, or (at your option) any later version.

Krossboard is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR
PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with Krossboard.
If not, see <https://www.gnu.org/licenses/>.
*/

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
		Convey("Given a new instance of UsageDB", func() {
			usageDb := NewUsageDb(dbName, 100)

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
							CPUUsage: []*ResourceUsageItem{
								{
									DateUTC: RoundTime(start.Add(time.Duration(5*2)*time.Minute), time.Duration(usageDb.Step)*time.Second),
									Value:   10.066666666666666,
								},
								{
									DateUTC: RoundTime(start.Add(time.Duration(5*3)*time.Minute), time.Duration(usageDb.Step)*time.Second),
									Value:   20.066666666666666,
								},
							},
							MEMUsage: []*ResourceUsageItem{
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
					// TODO: make it work in CI
					//{
					//	name: "monthly test case - nominal",
					//	input: input{
					//		fetcher:  (*UsageDb).FetchUsageMonthly,
					//		duration: time.Duration(2664000) * time.Second * 2, // 2 months
					//		data: func() []data {
					//			someValues := []float64{60.466029, 94.050909, 66.456005, 43.771419, 42.463750, 68.682307, 6.563702, 15.651925, 9.696952, 30.091186}
					//
					//			start := 5    // minutes
					//			step := 5     // minutes
					//			end := 131400 // minutes ~ 3 months
					//
					//			var res []data
					//			i := 0
					//			for t := start; t < end; t += step {
					//				cpuUsage := someValues[(i % 10)]
					//				i++
					//				memUsage := someValues[(i % 10)]
					//				i++
					//
					//				res = append(res, data{
					//					t:        t,
					//					cpuUsage: cpuUsage,
					//					memUsage: memUsage})
					//			}
					//
					//			return res
					//		},
					//	},
					//	want: &UsageHistory{
					//		CPUUsage: []*ResourceUsageItem{
					//			{
					//				DateUTC: time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC),
					//				Value:   2752.122998720052,
					//			},
					//			{
					//				DateUTC: time.Date(start.Year(), start.Month()+1, 1, 0, 0, 0, 0, time.UTC),
					//				Value:   27661.31926200003,
					//			},
					//			{
					//				DateUTC: time.Date(start.Year(), start.Month()+2, 1, 0, 0, 0, 0, time.UTC),
					//				Value:   24540.338306223366,
					//			},
					//		},
					//		MEMUsage: []*ResourceUsageItem{
					//			{
					//				DateUTC: time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC),
					//				Value:   3736.6572568958586,
					//			},
					//			{
					//				DateUTC: time.Date(start.Year(), start.Month()+1, 1, 0, 0, 0, 0, time.UTC),
					//				Value:   37584.91415399974,
					//			},
					//			{
					//				DateUTC: time.Date(start.Year(), start.Month()+2, 1, 0, 0, 0, 0, time.UTC),
					//				Value:   33345.75017615485,
					//			},
					//		},
					//	},
					//},
				}

				for _, test := range tests {
					Convey(fmt.Sprintf("Given the test case '%s'", test.name), func() {
						Convey("Given some values added in the instance of RRD", func() {
							data := test.input.data()

							for _, datum := range data {
								err = usageDb.UpdateRRD(start.Add(time.Duration(datum.t)*time.Minute), datum.cpuUsage, datum.memUsage)
								if err != nil {
									// check only for faulty step (to limit the number of assertions)
									So(err, ShouldBeNil)
								}
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

func TestComputeCumulativeMonth(t *testing.T) {
	Convey("Given a a set of history data", t, func(c C) {
		type args struct {
			items []*ResourceUsageItem
		}
		tests := []struct {
			name string
			args args
			want []*ResourceUsageItem
		}{
			{
				name: "empty",
				args: args{
					items: []*ResourceUsageItem{
					},
				},
				want: []*ResourceUsageItem{
				},
			},
			{
				name: "1 month",
				args: args{
					items: []*ResourceUsageItem{
						{DateUTC: date(c, "2020-01-02T15:14:05Z"), Value: 10},
						{DateUTC: date(c, "2020-01-12T16:24:05Z"), Value: 20},
						{DateUTC: date(c, "2020-01-24T17:34:05Z"), Value: 30},
						{DateUTC: date(c, "2020-01-30T18:44:05Z"), Value: 40},
					},
				},
				want: []*ResourceUsageItem{
					{DateUTC: date(c, "2020-01-01T00:00:00Z"), Value: 100},
				},
			},
			{
				name: "2 months (same year)",
				args: args{
					items: []*ResourceUsageItem{
						{DateUTC: date(c, "2020-01-02T15:14:05Z"), Value: 10},
						{DateUTC: date(c, "2020-01-12T16:24:05Z"), Value: 20},
						{DateUTC: date(c, "2020-02-14T17:34:05Z"), Value: 30},
						{DateUTC: date(c, "2020-02-28T18:44:05Z"), Value: 40},
					},
				},
				want: []*ResourceUsageItem{
					{DateUTC: date(c, "2020-01-01T00:00:00Z"), Value: 30},
					{DateUTC: date(c, "2020-02-01T00:00:00Z"), Value: 70},
				},
			},
			{
				name: "3 months (2 years)",
				args: args{
					items: []*ResourceUsageItem{
						{DateUTC: date(c, "2020-12-02T15:14:05Z"), Value: 10},
						{DateUTC: date(c, "2020-12-12T16:24:05Z"), Value: 20},
						{DateUTC: date(c, "2021-01-14T17:34:05Z"), Value: 30},
						{DateUTC: date(c, "2021-02-28T18:44:05Z"), Value: 40},
					},
				},
				want: []*ResourceUsageItem{
					{DateUTC: date(c, "2020-12-01T00:00:00Z"), Value: 30},
					{DateUTC: date(c, "2021-01-01T00:00:00Z"), Value: 30},
					{DateUTC: date(c, "2021-02-01T00:00:00Z"), Value: 40},
				},
			},
		}
		for _, tt := range tests {
			Convey(fmt.Sprintf("When performing the consolidation for '%s'", tt.name), func() {
				got := computeCumulativeMonth(tt.args.items)

				Convey("Then monthly consolidated values are the expected ones", func() {
					So(got, ShouldResemble, tt.want)
				})
			})
		}
	})
}

func date(c C, dateStr string) time.Time {
	t, err := time.Parse(time.RFC3339, dateStr)

	So(err, ShouldBeNil)

	return t
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
