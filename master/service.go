package master

import (
	"sync"

	"github.com/baidu/openedge/master/engine"
	"github.com/baidu/openedge/sdk-go/openedge"
	"github.com/baidu/openedge/utils"
	"github.com/docker/distribution/uuid"
)

// Auth auth api request from services
func (m *Master) Auth(username, password string) bool {
	v, ok := m.accounts.Get(username)
	if !ok {
		return false
	}
	p, ok := v.(string)
	return ok && p == password
}

func (m *Master) initServices() error {
	if utils.FileExists(configFile) {
		curcfg := new(DynamicConfig)
		err := utils.LoadYAML(configFile, curcfg)
		if err != nil {
			return err
		}
		m.curcfg = curcfg
		return m.startServices(m.curcfg.Services)
	}
	return m.startServices(m.inicfg.Services)
}

func (m *Master) stopAllServices() {
	var wg sync.WaitGroup
	for _, s := range m.services.Items() {
		wg.Add(1)
		go func(s engine.Service) {
			defer wg.Done()
			s.Stop()
			m.services.Remove(s.Name())
			m.accounts.Remove(s.Name())
		}(s.(engine.Service))
	}
	wg.Wait()
}

func (m *Master) startServices(ss []engine.ServiceInfo) error {
	for _, s := range ss {
		cur, ok := m.services.Get(s.Name)
		if ok {
			cur.(engine.Service).Stop()
		}
		token := uuid.Generate().String()
		m.accounts.Set(s.Name, token)
		s.Env[openedge.EnvServiceNameKey] = s.Name
		s.Env[openedge.EnvServiceTokenKey] = token
		nxt, err := m.engine.Run(s)
		if err != nil {
			return err
		}
		m.services.Set(s.Name, nxt)
	}
	return nil
}

func (m *Master) stopServices(ss []engine.ServiceInfo) {
	var wg sync.WaitGroup
	for _, s := range ss {
		cur, ok := m.services.Get(s.Name)
		if !ok {
			continue
		}
		wg.Add(1)
		go func(ss engine.Service) {
			defer wg.Done()
			ss.Stop()
			m.services.Remove(ss.Name())
			m.accounts.Remove(ss.Name())
		}(cur.(engine.Service))
	}
	wg.Wait()
}
