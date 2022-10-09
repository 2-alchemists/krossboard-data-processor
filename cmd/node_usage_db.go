/*
   Copyright (C) 2020  2ALCHEMISTS SAS.

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as
   published by the Free Software Foundation, either version 3 of the
   License, or (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package cmd

import (
	"fmt"
	"math"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// NodeUsage holds an instance of node usage as processed by kube-opex-analytics
type NodeUsage struct {
	DateUTC        time.Time `json:"dateUTC,omitempty"`
	Name           string    `json:"name,omitempty"`
	CPUCapacity    float64   `json:"cpuCapacity,omitempty"`
	CPUAllocatable float64   `json:"cpuAllocatable,omitempty"`
	CPUUsageByPods float64   `json:"cpuUsageByPods,omitempty"`
	MEMCapacity    float64   `json:"memCapacity,omitempty"`
	MEMAllocatable float64   `json:"memAllocatable,omitempty"`
	MEMUsageByPods float64   `json:"memUsageByPods,omitempty"`
	PodsUsage      []*struct {
		CPUUsage float64 `json:"cpuUsage,omitempty"`
		MEMUsage float64 `json:"memUsage,omitempty"`
	} `json:"podsRunning,omitempty"`
}

type NodeUsageDb struct {
	AllocatableDb *UsageDb
	CapacityDb    *UsageDb
	UsageByPodsDb *UsageDb
}

func NewNodeUsageDB(nodeName string) *NodeUsageDb {
	dbDir := viper.GetString("krossboard_data_dir")
	capacityDbPath := fmt.Sprintf("%s/.nodeusage_%s_capacity", dbDir, nodeName)
	allocatableDbPath := fmt.Sprintf("%s/.nodeusage_%s_allocatable", dbDir, nodeName)
	usageByPodsDbPath := fmt.Sprintf("%s/.nodeusage_%s_usage_by_pods", dbDir, nodeName)

	dbSet := &NodeUsageDb{
		CapacityDb:    NewUsageDb(capacityDbPath, math.MaxFloat64),
		AllocatableDb: NewUsageDb(allocatableDbPath, math.MaxFloat64),
		UsageByPodsDb: NewUsageDb(usageByPodsDbPath, math.MaxFloat64),
	}

	fileCreated := false
	_, err := os.Stat(capacityDbPath)
	if os.IsNotExist(err) {
		err := dbSet.CapacityDb.CreateRRD()
		if err != nil {
			log.WithError(err).Errorln("failed creating RRD file", capacityDbPath)
		} else {
			fileCreated = true
		}
	}

	_, err = os.Stat(allocatableDbPath)
	if os.IsNotExist(err) {
		err := dbSet.AllocatableDb.CreateRRD()
		if err != nil {
			log.WithError(err).Errorln("failed creating RRD file", allocatableDbPath)
		} else {
			fileCreated = true
		}
	}

	_, err = os.Stat(usageByPodsDbPath)
	if os.IsNotExist(err) {
		err := dbSet.UsageByPodsDb.CreateRRD()
		if err != nil {
			log.WithError(err).Errorln("failed creating RRD file", usageByPodsDbPath)
		} else {
			fileCreated = true
		}
	}

	if fileCreated {
		time.Sleep(time.Second) // otherwise update may fail with 'illegal attempt to update' error
	}
	return dbSet
}
