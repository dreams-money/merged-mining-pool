package rpc

import (
	"errors"
	"log"
	"time"
)

type Manager struct {
	chainName            string
	activeIndex          int
	clients              []*RPCClient
	primaryCheckInterval time.Duration
}

func MakeRPCManager(chainName string, nodes []Config, returnToPrimaryAfter string) Manager {
	m := Manager{}
	m.chainName = chainName
	m.clients = make([]*RPCClient, len(nodes))
	for i, node := range nodes {
		m.clients[i] = NewRPCClient(node.Name, node.URL, node.Username, node.Password, node.Timeout)
	}
	var err error
	m.primaryCheckInterval, err = time.ParseDuration(returnToPrimaryAfter)
	if err != nil {
		panic(err)
	}
	return m
}

func (manager *Manager) GetActiveClient() *RPCClient {
	return manager.clients[manager.activeIndex]
}

func (manager *Manager) CheckAndRecoverRPCs() error {
	if manager.GetActiveClient().Check() {
		return nil
	}
	err := manager.FindHealthyNode()
	if err != nil {
		return err
	}
	// Launch loop to eventually get us back to the primary node
	go func() {
		for {
			time.Sleep(manager.primaryCheckInterval)
			if manager.CheckPrimary() {
				manager.RestorePrimary()
				return
			}
		}
	}()

	return nil
}

func (m *Manager) RestorePrimary() {
	m.activeIndex = 0
}

func (m *Manager) FindHealthyNode() error {
	nodesLength := len(m.clients)
	nodesChecked := 0
	nowHealthy := false
	for !nowHealthy {
		nodesChecked++
		if nodesChecked > nodesLength {
			return errors.New("no healthy " + m.chainName + " nodes!")
		}
		m.nextNode()
		nowHealthy = m.checkActiveNodeHealth()
	}
	log.Printf("Now on node: %v\n", m.activeIndex)
	return nil
}

func (m *Manager) CheckPrimary() bool {
	return m.clients[0].Check()
}

func (m *Manager) GetIndex() int {
	return m.activeIndex
}

func (m *Manager) checkActiveNodeHealth() bool {
	return m.GetActiveClient().Check()
}

func (m *Manager) nextNode() {
	if m.activeIndex < len(m.clients)-1 {
		m.activeIndex++
	} else {
		m.activeIndex = 0
	}
}
