package mqtt

import (
	"fmt"
	"strings"

	"erp/internal/config"
)

func Tenant(cfg *config.Config) string {
	if cfg == nil || strings.TrimSpace(cfg.Mqtt.Tenant) == "" {
		return "default"
	}
	return strings.TrimSpace(cfg.Mqtt.Tenant)
}

func UserTopic(tenant string, userID int64) string {
	return fmt.Sprintf("erp/%s/user/%d/notify", tenant, userID)
}

func RoleTopic(tenant, role string) string {
	return fmt.Sprintf("erp/%s/role/%s/notify", tenant, strings.TrimSpace(role))
}

func WorkflowTopic(tenant, bizType string, bizID int64) string {
	return fmt.Sprintf("erp/%s/workflow/%s/%d", tenant, bizType, bizID)
}

func UserUsername(userID int64) string {
	return fmt.Sprintf("u%d", userID)
}

func UserClientID(cfg *config.Config, userID int64) string {
	prefix := "erp"
	tenant := "default"
	if cfg != nil {
		if cfg.Mqtt.ClientPrefix != "" {
			prefix = cfg.Mqtt.ClientPrefix
		}
		tenant = Tenant(cfg)
	}
	return fmt.Sprintf("%s-%s-u%d", prefix, tenant, userID)
}

func TopicAllowedForUser(tenant string, userID int64, roles []string, topic string) bool {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return false
	}
	userPrefix := fmt.Sprintf("erp/%s/user/%d/", tenant, userID)
	if strings.HasPrefix(topic, userPrefix) {
		return true
	}
	for _, r := range roles {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		rp := fmt.Sprintf("erp/%s/role/%s/", tenant, r)
		if strings.HasPrefix(topic, rp) {
			return true
		}
	}
	return false
}
