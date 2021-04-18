package cmd

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"encoding/json"
)



func TestKOAClusterNodesUsage(t *testing.T) {
	nodesUsageDataset := []byte(`{
  "gke-cluster-1-default-pool-7f5e6673-lxjd": {
    "id": "83dc93c4-6941-428d-b7da-2b54de83317c",
    "name": "gke-cluster-1-default-pool-7f5e6673-lxjd",
    "state": "Ready",
    "message": "kubelet is posting ready status. AppArmor enabled",
    "cpuCapacity": 2,
    "cpuAllocatable": 0.9400000000000001,
    "cpuUsage": 0.098925039,
    "memCapacity": 4140904448,
    "memAllocatable": 2967547904,
    "memUsage": 781180928,
    "containerRuntime": "docker://19.3.6",
    "podsRunning": [
      {
        "id": "267ff19e-b190-448d-abcb-e64e677c7946",
        "name": "event-exporter-gke-666b7ffbf7-qdnvk.kube-system",
        "nodeName": "gke-cluster-1-default-pool-7f5e6673-lxjd",
        "phase": "Running",
        "state": "Initialized",
        "cpuUsage": 0.00031655600000000006,
        "memUsage": 15581184.0
      },
      {
        "id": "aae1824e-8a2d-48e1-9425-4ef44569948b",
        "name": "fluentbit-gke-gkhqm.kube-system",
        "nodeName": "gke-cluster-1-default-pool-7f5e6673-lxjd",
        "phase": "Running",
        "state": "Initialized",
        "cpuUsage": 0.002141337,
        "memUsage": 19709952.0
      },
      {
        "id": "07be6122-30cc-4293-9b73-a3cda22ceca0",
        "name": "gke-metrics-agent-vs7lk.kube-system",
        "nodeName": "gke-cluster-1-default-pool-7f5e6673-lxjd",
        "phase": "Running",
        "state": "Initialized",
        "cpuUsage": 0.00041653,
        "memUsage": 25079808.0
      },
      {
        "id": "596173b9-78ad-435e-a1ee-b72088150be9",
        "name": "kube-dns-9c59558bb-fr44k.kube-system",
        "nodeName": "gke-cluster-1-default-pool-7f5e6673-lxjd",
        "phase": "Running",
        "state": "Initialized",
        "cpuUsage": 0.002351143,
        "memUsage": 34996224.0
      },
      {
        "id": "34e9c698-3684-4453-a4d8-8d9f857cfa7d",
        "name": "kube-dns-autoscaler-5c78d65cd9-5h6sm.kube-system",
        "nodeName": "gke-cluster-1-default-pool-7f5e6673-lxjd",
        "phase": "Running",
        "state": "Initialized",
        "cpuUsage": 0.00017146000000000002,
        "memUsage": 14340096.0
      },
      {
        "id": "ec943ef0-d74a-42ab-9031-67d7f3f5f865",
        "name": "kube-proxy-gke-cluster-1-default-pool-7f5e6673-lxjd.kube-system",
        "nodeName": "gke-cluster-1-default-pool-7f5e6673-lxjd",
        "phase": "Running",
        "state": "Initialized",
        "cpuUsage": 0.0007400690000000001,
        "memUsage": 27795456.0
      },
      {
        "id": "675d2771-b0f0-427b-ade2-048fff0ff1a1",
        "name": "l7-default-backend-5b76b455d-vr5p4.kube-system",
        "nodeName": "gke-cluster-1-default-pool-7f5e6673-lxjd",
        "phase": "Running",
        "state": "Initialized",
        "cpuUsage": 0.00015354800000000001,
        "memUsage": 2469888.0
      },
      {
        "id": "8eeefa9b-8997-4a2f-9124-212f20797817",
        "name": "metrics-server-v0.3.6-547dc87f5f-4tcmq.kube-system",
        "nodeName": "gke-cluster-1-default-pool-7f5e6673-lxjd",
        "phase": "Running",
        "state": "Initialized",
        "cpuUsage": 0.000542879,
        "memUsage": 18952192.0
      },
      {
        "id": "11a73d08-c08b-4da5-8d63-b2e67c6f5feb",
        "name": "prometheus-to-sd-qfxcc.kube-system",
        "nodeName": "gke-cluster-1-default-pool-7f5e6673-lxjd",
        "phase": "Running",
        "state": "Initialized",
        "cpuUsage": 3.592e-05,
        "memUsage": 9961472.0
      },
      {
        "id": "784423d6-c88e-4ee1-91c6-a9479f1b5db4",
        "name": "stackdriver-metadata-agent-cluster-level-647ddb674b-4rpfl.kube-system",
        "nodeName": "gke-cluster-1-default-pool-7f5e6673-lxjd",
        "phase": "Running",
        "state": "Initialized",
        "cpuUsage": 0.020663815000000002,
        "memUsage": 25505792.0
      }
    ],
    "podsNotRunning": [
      
    ]
  }
}`)

	Convey("Test KOANodesUsage", t, func() {

		Convey("Given a valid dataset", func() {
			nodesUsage := &map[string]NodeUsage{}
			err := json.Unmarshal(nodesUsageDataset, nodesUsage)
			So(err, ShouldBeNil)
			So(len(*nodesUsage), ShouldEqual, 1)
			node, found := (*nodesUsage)["gke-cluster-1-default-pool-7f5e6673-lxjd"]
			So(found, ShouldBeTrue)
			So(node.Name, ShouldEqual, "gke-cluster-1-default-pool-7f5e6673-lxjd")
			So(node.CPUCapacity, ShouldEqual, 2)
			So(node.CPUAllocatable, ShouldEqual, 0.9400000000000001)
			So(node.MEMCapacity, ShouldEqual, 4140904448)
			So(node.MEMAllocatable, ShouldEqual, 2967547904)
		})
	})
}
