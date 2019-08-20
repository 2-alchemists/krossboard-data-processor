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

// NewSystemStatus creates a new system status management object
func NewSystemStatus(statusFile string) *SystemStatus {
	return &SystemStatus{
		StatusFile: statusFile,
	}
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
