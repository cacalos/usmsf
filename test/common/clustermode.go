package common

import "strings"

// ClusterMode Cluster 구동 Mode
type ClusterMode int32

const (
	//ClusterStandAlone 단독 기동
	ClusterStandAlone ClusterMode = iota
	//ClusterActor Proto-Actor + CONSUL을 사용한 Actor Mode Cluster 사용
	ClusterActor
	//ClusterK8S RESTFUL + Kubernetes를 사용한 Container Mode Cluster 사용
	ClusterK8S
)

//StringToClusterMode ClusterMode를 반환한다.
func StringToClusterMode(cm string) ClusterMode {
	if strings.EqualFold(cm, "actor") {
		return ClusterActor
	} else if strings.EqualFold(cm, "kubernetes") {
		return ClusterK8S
	} else if strings.EqualFold(cm, "k8s") {
		return ClusterK8S
	}
	return ClusterStandAlone
}
