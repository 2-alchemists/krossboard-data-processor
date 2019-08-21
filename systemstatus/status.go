package systemstatus

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/rchakode/kube-opex-analytics-mc/koainstance"
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

// LoadInstanceSet loads instance status from file
func (m *SystemStatus) LoadInstanceSet() (*InstanceSet, error) {
	content, err := ioutil.ReadFile(m.StatusFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed reading file %v", m.StatusFile)
	}
	systemStatus := &InstanceSet{}
	err = json.Unmarshal(content, systemStatus)
	if err != nil {
		return nil, errors.Wrapf(err, "failed decoding status file content")
	}

	return systemStatus, nil
}

// InitializeStatusIfEmpty initializes the status file with empty instance list
func (m *SystemStatus) InitializeStatusIfEmpty() error {
	_, err := os.Stat(m.StatusFile)
	if os.IsNotExist(err) {
		return m.UpdateInstanceSet(&InstanceSet{
			NextHostPort: 49000,
			Instances:    []*koainstance.Instance{},
		})
	}
	return err
}

// UpdateInstanceSet update system status with given instance data
func (m *SystemStatus) UpdateInstanceSet(instanceSet *InstanceSet) error {
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

// FindInstance finds if there is an instance for a given K8s context
func (m *SystemStatus) FindInstance(clusterContext string) (int, error) {
	instanceSet, err := m.LoadInstanceSet()
	if err != nil {
		return -1, errors.Wrap(err, "failed loading instance set")
	}

	foundIndex := int(-1)
	for index, instance := range instanceSet.Instances {
		if instance.ClusterContext == clusterContext {
			foundIndex = index
			break
		}
	}
	return foundIndex, nil
}
