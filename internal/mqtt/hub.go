package mqtt

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"erp/internal/alert"
	"erp/internal/config"
)

// Hub publishes workflow notifications via NanoMQ using server credentials.
type Hub struct {
	cfg    *config.Config
	mu     sync.Mutex
	client mqtt.Client
}

func NewHub(cfg *config.Config) *Hub {
	return &Hub{cfg: cfg}
}

func (h *Hub) Connected() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.client != nil && h.client.IsConnected()
}

func (h *Hub) ensureConnected() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cfg == nil || !h.cfg.Mqtt.Enabled {
		return nil
	}
	if h.client != nil && h.client.IsConnected() {
		return nil
	}
	opts := mqtt.NewClientOptions()
	opts.AddBroker(h.cfg.Mqtt.BrokerURL)
	opts.SetClientID(h.cfg.Mqtt.ClientPrefix + "-hub")
	opts.SetUsername(h.cfg.Mqtt.ServerUsername)
	opts.SetPassword(h.cfg.Mqtt.ServerPassword)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(3 * time.Second)
	opts.SetKeepAlive(time.Duration(h.cfg.Mqtt.KeepAliveSeconds) * time.Second)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if ok := token.WaitTimeout(8 * time.Second); !ok || token.Error() != nil {
		err := token.Error()
		if err == nil {
			err = errConnectTimeout
		}
		log.Printf("mqtt hub connect: %v", err)
		alert.Default.Warn("mqtt", err.Error())
		return err
	}
	h.client = client
	return nil
}

var errConnectTimeout = errString("mqtt connect timeout")

type errString string

func (e errString) Error() string { return string(e) }

func (h *Hub) Publish(topic string, payload interface{}) error {
	if h.cfg == nil || !h.cfg.Mqtt.Enabled {
		return nil
	}
	if err := h.ensureConnected(); err != nil {
		return err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	h.mu.Lock()
	client := h.client
	h.mu.Unlock()
	tok := client.Publish(topic, 1, false, b)
	if ok := tok.WaitTimeout(5 * time.Second); !ok || tok.Error() != nil {
		return tok.Error()
	}
	return nil
}

func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.client != nil && h.client.IsConnected() {
		h.client.Disconnect(250)
	}
}
