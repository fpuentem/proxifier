package proxy

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	redis "gopkg.in/redis.v5"

	"github.com/rookmoot/proxifier/logger"
)

// RedisInterface defines the interface for interacting with Redis.
type RedisInterface interface {
	SMembers(key string) *redis.StringSliceCmd
	SAdd(key string, members ...interface{}) *redis.IntCmd
	HMGet(key string, fields ...string) *redis.SliceCmd
	HMSet(key string, fields map[string]string) *redis.StatusCmd
	HSet(key, field string, value interface{}) *redis.BoolCmd
	HGet(key, field string) *redis.StringCmd
	Incr(key string) *redis.IntCmd
}

// Manager manages proxies and provides proxy rotation functionality.
type Manager struct {
	db      RedisInterface
	log     logger.Logger
	proxies []*Proxy
}

// NewManager creates a new Manager instance.
func NewManager(db RedisInterface, log logger.Logger) (*Manager, error) {
	m := Manager{
		db:  db,
		log: log,
	}

	err := m.loadProxyList()
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// UpdateProxies updates the list of proxies from a JSON file.
func (m *Manager) UpdateProxies(filepath string) error {
	proxies, err := m.readProxiesFromFile(filepath)
	if err != nil {
		return err
	}

	for _, proxy := range proxies {
		if proxy.GetAnonymityLevel() == "elite" && (proxy.GetProtocol() == "http" || proxy.GetProtocol() == "https") {
			if m.proxyExists(proxy) == false {
				m.log.Info("proxy: %v (%v, %v)", proxy.GetAddress(), proxy.GetProtocol(), proxy.GetAnonymityLevel())
				err = m.proxySave(proxy)
				if err != nil {
					m.log.Warn("proxy: %v", err)
				}
			}
		}
	}

	err = m.loadProxyList()
	if err != nil {
		return err
	}

	return nil
}

// GetProxy returns a random proxy from the list.
func (m *Manager) GetProxy() (*Proxy, error) {
	// rand.Seed(time.Now().Unix())
	r := rand.New(rand.NewSource(time.Now().Unix()))
	vr := r.Intn(len(m.proxies))
	return m.proxies[vr], nil
}

// loadProxyList loads proxies from Redis and populates the Manager's proxy list.
func (m *Manager) loadProxyList() error {
	ret, err := m.db.SMembers("proxies").Result()
	if err != nil {
		return err
	}

	for _, v := range ret {
		pid, _ := strconv.Atoi(v)
		p, err := m.loadProxy(pid)
		if err != nil {
			m.log.Warn("proxy: %v", err)
		} else {
			m.proxies = append(m.proxies, p)
		}
	}

	return nil
}

// loadProxy loads a single proxy from Redis based on its ID.
func (m *Manager) loadProxy(pid int) (*Proxy, error) {
	data, err := m.db.HMGet(fmt.Sprintf("proxy:%d", pid), "ipaddress", "port", "protocol", "anonymitylevel", "source", "country").Result()
	if err != nil {
		return nil, err
	}

	infos := make(map[string]string, 6)
	infos["ipaddress"] = data[0].(string)
	infos["port"] = data[1].(string)
	infos["protocol"] = data[2].(string)
	infos["anonymitylevel"] = data[3].(string)
	infos["source"] = data[4].(string)
	infos["country"] = data[5].(string)

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", infos["ipaddress"], infos["port"]))
	if err != nil {
		return nil, err
	}

	p := Proxy{
		id:    pid,
		addr:  addr,
		infos: infos,
	}
	return &p, nil
}

// proxySave saves a proxy to Redis.
func (m *Manager) proxySave(p Proxy) error {
	next_id, err := m.db.Incr("proxies_next_id").Result()
	if err != nil {
		return err
	}

	_, err = m.db.HMSet(fmt.Sprintf("proxy:%d", next_id), p.infos).Result()
	if err != nil {
		return err
	}

	_, err = m.db.SAdd("proxies", next_id).Result()
	if err != nil {
		return err
	}

	_, err = m.db.HSet("proxies_ids", p.GetAddress(), next_id).Result()
	if err != nil {
		return err
	}
	return nil
}

// proxyExists checks if a proxy exists in Redis.
func (m *Manager) proxyExists(p Proxy) bool {
	_, err := m.db.HGet("proxies_ids", p.GetAddress()).Result()
	if err != nil {
		return false
	}
	return true
}

// readProxiesFromFile reads proxy configurations from a JSON file.
func (m *Manager) readProxiesFromFile(filepath string) ([]Proxy, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var proxies []Proxy
	var values []map[string]interface{}

	err = json.Unmarshal([]byte(file), &values)
	if err != nil {
		return nil, err
	}

	for _, data := range values {
		infos := make(map[string]string, 6)
		for k, v := range data {
			if k == "protocols" {
				for _, tmp := range v.([]interface{}) {
					infos["protocol"] = tmp.(string)
				}
			} else if k == "port" {
				infos["port"] = fmt.Sprintf("%v", v)
			} else {
				infos[strings.ToLower(k)] = strings.ToLower(v.(string))
			}
		}

		p := Proxy{
			id:    0,
			addr:  nil,
			infos: infos,
		}
		proxies = append(proxies, p)
	}

	return proxies, nil
}
