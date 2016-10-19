package k8s

import ()

// Trigger indicates why the event was triggered
type Trigger string

const (
	POD_ADDED           Trigger = "POD_ADDED"
	POD_MODIFIED                = "POD_MODIFIED"
	POD_DELETED                 = "POD_DELETED"
	CONFIG_MAP_ADDED            = "CONFIG_MAP_ADDED"
	CONFIG_MAP_MODIFIED         = "CONFIG_MAP_MODIFIED"
	CONFIG_MAP_DELETED          = "CONFIG_MAP_DELETED"
)

// NodeEvent indicates when something changed with the node (either a pod or config map changed)
type NodeEvent struct {
	Trigger
	*Pod
	*ConfigMapEntry
}
