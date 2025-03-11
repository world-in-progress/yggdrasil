package scene

import (
	"fmt"

	componentinterface "github.com/world-in-progress/yggdrasil/component/interface"
	"github.com/world-in-progress/yggdrasil/core/threading"
	"github.com/world-in-progress/yggdrasil/node"
)

// SyncTask is the structure for a synchronously call a specific node and its component
type SyncTask struct {
	threading.BaseTask
	Result  chan any
	ERR     chan error
	headers map[string]string
	params  map[string]any
	tree    *node.Tree
	node    *node.Node
	compo   componentinterface.IComponent
}

func NewSyncTask(taskID string, tree *node.Tree, node *node.Node, compo componentinterface.IComponent, params map[string]any, headers map[string]string) *SyncTask {

	task := &SyncTask{
		BaseTask: threading.BaseTask{
			ID: taskID,
		},
		Result:  make(chan any, 1),
		ERR:     make(chan error, 1),
		tree:    tree,
		node:    node,
		compo:   compo,
		params:  params,
		headers: headers,
	}

	return task
}

func (st *SyncTask) Process() {
	result, err := st.compo.Execute(st.node, st.params, nil, st.headers)
	if err != nil {
		st.ERR <- fmt.Errorf("error executing component %v of node %v: %v",
			st.compo.GetName(), st.node.GetName(), err)
		return
	}
	st.Result <- result
}

func (st *SyncTask) Syncing() (any, error) {
	select {
	case result := <-st.Result:
		// update node attribute if the attribute name is provided in the result
		r := result.(map[string]any)
		for attribute, value := range r {
			st.tree.UpdateNodeAttribute(st.node.GetID(), attribute, value)
		}
		return result, nil
	case err := <-st.ERR:
		return nil, err
	}
}
