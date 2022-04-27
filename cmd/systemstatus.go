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
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

// InstanceSet hold the current status of an installation
type InstanceSet struct {
	NextHostPort int64       `json:"nextHostPort"`
	Instances    []*Instance `json:"instances"`
}

// SystemStatus hold the system status file
type SystemStatus struct {
	StatusFile string `json:"statusFile"`
}

// LoadSystemStatus creates a new system status object and load new or existing system status
func LoadSystemStatus(statusFile string) (*SystemStatus, error) {
	systemStatus := &SystemStatus{
		StatusFile: statusFile,
	}
	err := systemStatus.InitializeStatusIfEmpty()
	return systemStatus, err
}

// GetInstances loads instances from status file
func (m *SystemStatus) GetInstances() (*InstanceSet, error) {
	content, err := ioutil.ReadFile(m.StatusFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed reading file %v", m.StatusFile)
	}
	instances := &InstanceSet{}
	err = json.Unmarshal(content, instances)
	if err != nil {
		return nil, errors.Wrapf(err, "failed decoding status file content")
	}

	return instances, nil
}

// GetInstance returns the info of an instance given its name or error otherwise
func (m *SystemStatus) GetInstance(clusterName string) (*Instance, error) {
	runningConfig, err := m.GetInstances()
	if err != nil {
		return nil, errors.Wrap(err, "failed fetching running configuration")
	}
	var resultInstance *Instance = nil
	for _, instance := range runningConfig.Instances {
		if instance.ClusterName == clusterName {
			resultInstance = instance
			break
		}
	}
	if resultInstance == nil {
		return nil, errors.New("cluster not found " + clusterName)
	}
	return resultInstance, nil
}

// InitializeStatusIfEmpty initializes the status file with empty instance list
func (m *SystemStatus) InitializeStatusIfEmpty() error {
	_, err := os.Stat(m.StatusFile)
	if os.IsNotExist(err) {
		return m.UpdateRunningConfig(&InstanceSet{
			NextHostPort: 49000,
			Instances:    []*Instance{},
		})
	}
	return err
}

// UpdateRunningConfig update system status with given instance data set
func (m *SystemStatus) UpdateRunningConfig(instanceSet *InstanceSet) error {
	content, err := json.Marshal(&instanceSet)
	if err != nil {
		return errors.Wrapf(err, "failed marshaling status object")
	}
	err = ioutil.WriteFile(m.StatusFile, content, 0600)
	if err != nil {
		return errors.Wrapf(err, "failed writing file %v", m.StatusFile)
	}
	return nil
}

// FindInstance finds if there is an instance for a given K8s name and return its ID
func (m *SystemStatus) FindInstance(clusterName string) (instanceID string, err error) {
	instanceID = ""
	runningConfig, err := m.GetInstances()
	if err != nil {
		return "", errors.Wrap(err, "failed fetching running configuration")
	}

	for _, instance := range runningConfig.Instances {
		if instance.ClusterName == clusterName {
			instanceID = instance.ID
			break
		}
	}
	return instanceID, nil
}

// RemoveInstanceByContainerID removes an instance mathing a container's ID
func (m *SystemStatus) RemoveInstanceByContainerID(containerID string) error {
	runningConfig, err := m.GetInstances()
	if err != nil {
		return errors.Wrap(err, "failed fetching running configuration")
	}

	newRunningConfig := &InstanceSet{
		NextHostPort: runningConfig.NextHostPort,
	}
	foundItemIndex := int(-1)
	for index, instance := range runningConfig.Instances {
		if instance.ID == containerID {
			foundItemIndex = index
		} else {
			newRunningConfig.Instances = append(newRunningConfig.Instances, instance)
		}
	}

	if foundItemIndex != -1 {
		return m.UpdateRunningConfig(newRunningConfig)
	}
	return nil
}
