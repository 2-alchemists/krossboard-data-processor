package systemstatus

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"bitbucket.org/koamc/kube-opex-analytics-mc/koainstance"
	"github.com/pkg/errors"
)

// InstanceSet hold the current status of an installation
type InstanceSet struct {
	NextHostPort int64                   `json:"nextHostPort"`
	Instances    []*koainstance.Instance `json:"instances"`
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

// InitializeStatusIfEmpty initializes the status file with empty instance list
func (m *SystemStatus) InitializeStatusIfEmpty() error {
	_, err := os.Stat(m.StatusFile)
	if os.IsNotExist(err) {
		return m.UpdateRunningConfig(&InstanceSet{
			NextHostPort: 49000,
			Instances:    []*koainstance.Instance{},
		})
	}
	return err
}

// UpdateRunningConfig update system status with given instance data set
func (m *SystemStatus) UpdateRunningConfig(instanceSet *InstanceSet) error {
	content, err := json.Marshal(&instanceSet)
	if err != nil {
		return errors.Wrapf(err, "failed marhsaling status object")
	}
	err = ioutil.WriteFile(m.StatusFile, content, 0600)
	if err != nil {
		return errors.Wrapf(err, "failed writing file %v", m.StatusFile)
	}
	return nil
}

// FindInstance finds if there is an instance for a given K8s name
func (m *SystemStatus) FindInstance(clusterName string) (int, error) {
	runningConfig, err := m.GetInstances()
	if err != nil {
		return -1, errors.Wrap(err, "failed fetching running configuration")
	}

	foundItemIndex := int(-1)
	for index, instance := range runningConfig.Instances {
		if instance.ClusterName == clusterName {
			foundItemIndex = index
			break
		}
	}
	return foundItemIndex, nil
}

// RemoveInstanceByContainerID removes an instance mathing a container's ID
func (m *SystemStatus) RemoveInstanceByContainerID(containerID string) (*InstanceSet, error) {
	runningConfig, err := m.GetInstances()
	if err != nil {
		return nil, errors.Wrap(err, "failed fetching running configuration")
	}

	newRunningConfig := &InstanceSet{
		NextHostPort : runningConfig.NextHostPort,
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
		m.UpdateRunningConfig(newRunningConfig)
	}
    return m.GetInstances()
}