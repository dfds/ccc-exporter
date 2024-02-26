package model

import "fmt"

type ClusterId string

const (
	ClusterIdProd                       ClusterId = "lkc-4npj6"
	ClusterIdDev                        ClusterId = "lkc-3wqzw"
	ClusterIdDevelopmentCluster4        ClusterId = "lkc-3m912"
	ClusterIdConfluentControlCenterTest ClusterId = "lkc-qykkd"
	ClusterIdSSUDevEnvironment          ClusterId = "lkc-pj37pk"
	ClusterIdSSUDevEnvironment2         ClusterId = "lkc-j9gknw"
)

var ConfluentClusters = []ClusterId{ClusterIdProd, ClusterIdDev, ClusterIdDevelopmentCluster4, ClusterIdSSUDevEnvironment, ClusterIdSSUDevEnvironment2}

func TryParse(s string) (ClusterId, error) {
	for _, cluster := range ConfluentClusters {
		if s == string(cluster) {
			return cluster, nil
		}
	}
	return "", fmt.Errorf("invalid cluster id: %s", s)
}
