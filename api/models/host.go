package models

import (
	"database/sql/driver"
	"devops-manager/api/protobuf"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// HostStatus 主机状态枚举
type HostStatus string

const (
	HostStatusPending  HostStatus = "pending"  // 待准入
	HostStatusApproved HostStatus = "approved" // 已准入
	HostStatusRejected HostStatus = "rejected" // 已拒绝
)

// Host 主机模型
type Host struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	HostID    string         `json:"host_id" gorm:"uniqueIndex;size:255;not null;comment:主机唯一标识"`
	Hostname  string         `json:"hostname" gorm:"size:255;not null;comment:主机名"`
	IP        string         `json:"ip" gorm:"size:45;comment:IP地址"`
	OS        string         `json:"os" gorm:"size:100;comment:操作系统"`
	Status    HostStatus     `json:"status" gorm:"size:20;default:pending;comment:主机状态"`
	Tags      JSON           `json:"tags" gorm:"type:json;comment:标签信息"`
	LastSeen  time.Time      `json:"last_seen" gorm:"comment:最后上报时间"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// JSON 自定义类型用于处理 JSON 字段
type JSON map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSON)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		*j = make(JSON)
		return nil
	}
}

// Value 实现 driver.Valuer 接口
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// TableName 指定表名
func (Host) TableName() string {
	return "hosts"
}

// ToProtobuf 转换为 protobuf 格式
func (h *Host) ToProtobuf() map[string]interface{} {
	tags := make(map[string]string)
	for k, v := range h.Tags {
		if str, ok := v.(string); ok {
			tags[k] = str
		}
	}

	return map[string]interface{}{
		"id":        h.HostID,
		"hostname":  h.Hostname,
		"ip":        h.IP,
		"os":        h.OS,
		"status":    string(h.Status),
		"tags":      tags,
		"last_seen": h.LastSeen.Unix(),
	}
}

// FromProtobuf 从 protobuf 格式创建
func (h *Host) FromProtobuf(data map[string]interface{}) {
	if hostID, ok := data["id"].(string); ok {
		h.HostID = hostID
	}
	if hostname, ok := data["hostname"].(string); ok {
		h.Hostname = hostname
	}
	if ip, ok := data["ip"].(string); ok {
		h.IP = ip
	}
	if os, ok := data["os"].(string); ok {
		h.OS = os
	}
	if tags, ok := data["tags"].(map[string]string); ok {
		h.Tags = make(JSON)
		for k, v := range tags {
			h.Tags[k] = v
		}
	}
	if lastSeen, ok := data["last_seen"].(int64); ok {
		h.LastSeen = time.Unix(lastSeen, 0)
	}
}

// PendingHost 待准入主机信息（存储在 Redis 中）
type PendingHost struct {
	HostID    string            `json:"host_id"`
	Hostname  string            `json:"hostname"`
	IP        string            `json:"ip"`
	OS        string            `json:"os"`
	Tags      map[string]string `json:"tags"`
	FirstSeen int64             `json:"first_seen"` // 首次注册时间
	LastSeen  int64             `json:"last_seen"`  // 最后上报时间
}

// ToHost 转换为 Host 模型
func (ph *PendingHost) ToHost() *Host {
	tags := make(JSON)
	for k, v := range ph.Tags {
		tags[k] = v
	}

	return &Host{
		HostID:   ph.HostID,
		Hostname: ph.Hostname,
		IP:       ph.IP,
		OS:       ph.OS,
		Status:   HostStatusPending,
		Tags:     tags,
		LastSeen: time.Unix(ph.LastSeen, 0),
	}
}

// FromProtobufToPending 从 protobuf 创建待准入主机
func FromProtobufToPending(hostInfo *protobuf.HostInfo) *PendingHost {
	return &PendingHost{
		HostID:    hostInfo.Id,
		Hostname:  hostInfo.Hostname,
		IP:        hostInfo.Ip,
		OS:        hostInfo.Os,
		Tags:      hostInfo.Tags,
		FirstSeen: time.Now().Unix(),
		LastSeen:  hostInfo.LastSeen,
	}
}
