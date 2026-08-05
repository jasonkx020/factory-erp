package mqtt

import (
	"erp/internal/config"
	"erp/internal/security"
)

// IssueUserConnectInfo builds mqtt-connect payload for logged-in ERP user.
func IssueUserConnectInfo(cfg *config.Config, userID int64, roles []string) (map[string]interface{}, error) {
	if cfg == nil {
		cfg = &config.Config{}
	}
	tenant := Tenant(cfg)
	username := UserUsername(userID)
	clientID := UserClientID(cfg, userID)
	subs := []string{UserTopic(tenant, userID)}
	for _, r := range roles {
		if r == "" {
			continue
		}
		subs = append(subs, RoleTopic(tenant, r))
	}
	out := map[string]interface{}{
		"enabled":            cfg.Mqtt.Enabled,
		"broker_url":         cfg.Mqtt.BrokerURL,
		"ws_url":             cfg.Mqtt.WSURL,
		"client_id":          clientID,
		"username":           username,
		"keep_alive_seconds": cfg.Mqtt.KeepAliveSeconds,
		"qos_default":        1,
		"subscribe_topics":   subs,
	}
	if !cfg.Mqtt.Enabled {
		out["password"] = ""
		out["expires_in"] = 0
		return map[string]interface{}{"mqtt": out}, nil
	}
	token, exp, err := security.IssueMqttToken(cfg.JWT.Secret, cfg.Mqtt.TokenTTLSeconds, userID, username, clientID, tenant, roles)
	if err != nil {
		return nil, err
	}
	out["password"] = token
	out["expires_in"] = exp
	return map[string]interface{}{"mqtt": out}, nil
}
