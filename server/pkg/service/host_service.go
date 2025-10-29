package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"devops-manager/api/models"
	"devops-manager/api/protobuf"
	"devops-manager/server/pkg/database"

	"gorm.io/gorm"
)

// HostService 主机服务，提供统一的数据存储和访问
type HostService struct {
	db    *gorm.DB
	mutex sync.RWMutex
}

var (
	instance *HostService
	once     sync.Once
)

// GetHostService 获取主机服务单例
func GetHostService() *HostService {
	once.Do(func() {
		instance = &HostService{
			db: database.GetDB(),
		}
	})
	return instance
}

// RegisterHost 注册或更新主机信息
func (hs *HostService) RegisterHost(hostInfo *protobuf.HostInfo) error {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()

	// 如果没有ID，生成一个
	if hostInfo.Id == "" {
		hostInfo.Id = generateHostID()
	}

	// 查找现有主机（已准入的）
	var host models.Host
	result := hs.db.Where("host_id = ? AND status = ?", hostInfo.Id, models.HostStatusApproved).First(&host)

	now := time.Now()
	hostInfo.LastSeen = now.Unix()

	if result.Error == gorm.ErrRecordNotFound {
		// 主机不存在或未准入，检查是否在待准入列表中
		if hs.isPendingHost(hostInfo.Id) {
			// 更新待准入主机的信息
			return hs.updatePendingHost(hostInfo)
		} else {
			// 新主机，添加到待准入列表
			return hs.addToPendingList(hostInfo)
		}
	} else if result.Error != nil {
		return fmt.Errorf("failed to query host: %w", result.Error)
	} else {
		// 更新已准入主机的信息
		return hs.updateApprovedHost(&host, hostInfo)
	}
}

// GetHost 获取单个主机信息
func (hs *HostService) GetHost(id string) (*protobuf.HostInfo, bool) {
	hs.mutex.RLock()
	defer hs.mutex.RUnlock()

	// 先尝试从 Redis 缓存获取
	if hostInfo := hs.getCachedHost(id); hostInfo != nil {
		return hostInfo, true
	}

	// 从数据库获取
	var host models.Host
	if err := hs.db.Where("host_id = ?", id).First(&host).Error; err != nil {
		return nil, false
	}

	// 转换为 protobuf 格式
	hostInfo := hs.modelToProtobuf(&host)

	// 缓存到 Redis
	hs.cacheHost(hostInfo)

	return hostInfo, true
}

// GetAllHosts 获取所有主机信息（只返回已准入的）
func (hs *HostService) GetAllHosts() []*protobuf.HostInfo {
	hs.mutex.RLock()
	defer hs.mutex.RUnlock()

	var hosts []models.Host
	if err := hs.db.Where("status = ?", models.HostStatusApproved).Find(&hosts).Error; err != nil {
		return []*protobuf.HostInfo{}
	}

	result := make([]*protobuf.HostInfo, 0, len(hosts))
	for _, host := range hosts {
		hostInfo := hs.modelToProtobuf(&host)
		result = append(result, hostInfo)
	}
	return result
}

// UpdateHost 更新主机信息
func (hs *HostService) UpdateHost(hostInfo *protobuf.HostInfo) error {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()

	var host models.Host
	if err := hs.db.Where("host_id = ?", hostInfo.Id).First(&host).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrHostNotFound
		}
		return fmt.Errorf("failed to query host: %w", err)
	}

	// 准备标签数据
	tags := make(models.JSON)
	for k, v := range hostInfo.Tags {
		tags[k] = v
	}

	// 更新主机信息
	host.Hostname = hostInfo.Hostname
	host.IP = hostInfo.Ip
	host.OS = hostInfo.Os
	host.Tags = tags
	host.LastSeen = time.Now()

	if err := hs.db.Save(&host).Error; err != nil {
		return fmt.Errorf("failed to update host: %w", err)
	}

	// 更新时间戳
	hostInfo.LastSeen = host.LastSeen.Unix()

	// 更新缓存
	hs.cacheHost(hostInfo)

	return nil
}

// DeleteHost 删除主机
func (hs *HostService) DeleteHost(id string) error {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()

	// 软删除主机
	if err := hs.db.Where("host_id = ?", id).Delete(&models.Host{}).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrHostNotFound
		}
		return fmt.Errorf("failed to delete host: %w", err)
	}

	// 从缓存中删除
	hs.deleteCachedHost(id)

	return nil
}

// GetHostCount 获取主机统计信息
func (hs *HostService) GetHostCount() (total, online, offline int) {
	hs.mutex.RLock()
	defer hs.mutex.RUnlock()

	var count int64
	hs.db.Model(&models.Host{}).Count(&count)
	total = int(count)

	// 计算在线主机数（60秒内有心跳）
	var onlineCount int64
	hs.db.Model(&models.Host{}).Where("last_seen > ?", time.Now().Add(-60*time.Second)).Count(&onlineCount)
	online = int(onlineCount)
	offline = total - online

	return total, online, offline
}

func generateHostID() string {
	return "host-" + time.Now().Format("20060102150405")
}

// 错误定义
var (
	ErrHostNotFound = &HostError{Code: "HOST_NOT_FOUND", Message: "Host not found"}
)

type HostError struct {
	Code    string
	Message string
}

func (e *HostError) Error() string {
	return e.Message
}

// modelToProtobuf 将数据库模型转换为 protobuf 格式
func (hs *HostService) modelToProtobuf(host *models.Host) *protobuf.HostInfo {
	tags := make(map[string]string)
	for k, v := range host.Tags {
		if str, ok := v.(string); ok {
			tags[k] = str
		}
	}

	return &protobuf.HostInfo{
		Id:       host.HostID,
		Hostname: host.Hostname,
		Ip:       host.IP,
		Os:       host.OS,
		Tags:     tags,
		LastSeen: host.LastSeen.Unix(),
	}
}

// cacheHost 缓存主机信息到 Redis
func (hs *HostService) cacheHost(hostInfo *protobuf.HostInfo) {
	redis := database.GetRedis()
	if redis == nil {
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("host:%s", hostInfo.Id)

	data, err := json.Marshal(hostInfo)
	if err != nil {
		return
	}

	// 缓存5分钟
	redis.Set(ctx, key, data, 5*time.Minute)
}

// getCachedHost 从 Redis 获取缓存的主机信息
func (hs *HostService) getCachedHost(id string) *protobuf.HostInfo {
	redis := database.GetRedis()
	if redis == nil {
		return nil
	}

	ctx := context.Background()
	key := fmt.Sprintf("host:%s", id)

	data, err := redis.Get(ctx, key).Result()
	if err != nil {
		return nil
	}

	var hostInfo protobuf.HostInfo
	if err := json.Unmarshal([]byte(data), &hostInfo); err != nil {
		return nil
	}

	return &hostInfo
}

// deleteCachedHost 从 Redis 删除缓存的主机信息
func (hs *HostService) deleteCachedHost(id string) {
	redis := database.GetRedis()
	if redis == nil {
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("host:%s", id)
	redis.Del(ctx, key)
}

// addToPendingList 添加主机到待准入列表
func (hs *HostService) addToPendingList(hostInfo *protobuf.HostInfo) error {
	pendingHost := models.FromProtobufToPending(hostInfo)

	redis := database.GetRedis()
	if redis == nil {
		return fmt.Errorf("redis not available")
	}

	ctx := context.Background()
	key := fmt.Sprintf("pending_host:%s", hostInfo.Id)

	data, err := json.Marshal(pendingHost)
	if err != nil {
		return fmt.Errorf("failed to marshal pending host: %w", err)
	}

	// 存储到 Redis，不设置过期时间
	if err := redis.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to store pending host: %w", err)
	}

	return nil
}

// updatePendingHost 更新待准入主机信息
func (hs *HostService) updatePendingHost(hostInfo *protobuf.HostInfo) error {
	redis := database.GetRedis()
	if redis == nil {
		return fmt.Errorf("redis not available")
	}

	ctx := context.Background()
	key := fmt.Sprintf("pending_host:%s", hostInfo.Id)

	// 获取现有的待准入主机信息
	data, err := redis.Get(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to get pending host: %w", err)
	}

	var pendingHost models.PendingHost
	if err := json.Unmarshal([]byte(data), &pendingHost); err != nil {
		return fmt.Errorf("failed to unmarshal pending host: %w", err)
	}

	// 更新信息，保留首次注册时间
	pendingHost.Hostname = hostInfo.Hostname
	pendingHost.IP = hostInfo.Ip
	pendingHost.OS = hostInfo.Os
	pendingHost.Tags = hostInfo.Tags
	pendingHost.LastSeen = hostInfo.LastSeen

	// 重新存储
	newData, err := json.Marshal(pendingHost)
	if err != nil {
		return fmt.Errorf("failed to marshal updated pending host: %w", err)
	}

	return redis.Set(ctx, key, newData, 0).Err()
}

// updateApprovedHost 更新已准入主机信息
func (hs *HostService) updateApprovedHost(host *models.Host, hostInfo *protobuf.HostInfo) error {
	// 准备标签数据
	tags := make(models.JSON)
	for k, v := range hostInfo.Tags {
		tags[k] = v
	}

	// 更新主机信息
	host.Hostname = hostInfo.Hostname
	host.IP = hostInfo.Ip
	host.OS = hostInfo.Os
	host.Tags = tags
	host.LastSeen = time.Unix(hostInfo.LastSeen, 0)

	if err := hs.db.Save(host).Error; err != nil {
		return fmt.Errorf("failed to update approved host: %w", err)
	}

	// 缓存到 Redis
	hs.cacheHost(hostInfo)

	return nil
}

// isPendingHost 检查主机是否在待准入列表中
func (hs *HostService) isPendingHost(hostID string) bool {
	redis := database.GetRedis()
	if redis == nil {
		return false
	}

	ctx := context.Background()
	key := fmt.Sprintf("pending_host:%s", hostID)

	_, err := redis.Get(ctx, key).Result()
	return err == nil
}

// GetPendingHosts 获取所有待准入主机
func (hs *HostService) GetPendingHosts() ([]*models.PendingHost, error) {
	redis := database.GetRedis()
	if redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	ctx := context.Background()

	// 获取所有待准入主机的键
	keys, err := redis.Keys(ctx, "pending_host:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get pending host keys: %w", err)
	}

	var pendingHosts []*models.PendingHost
	for _, key := range keys {
		data, err := redis.Get(ctx, key).Result()
		if err != nil {
			continue // 跳过错误的键
		}

		var pendingHost models.PendingHost
		if err := json.Unmarshal([]byte(data), &pendingHost); err != nil {
			continue // 跳过解析错误的数据
		}

		pendingHosts = append(pendingHosts, &pendingHost)
	}

	return pendingHosts, nil
}

// ApproveHost 准入主机
func (hs *HostService) ApproveHost(hostID string) error {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()

	redis := database.GetRedis()
	if redis == nil {
		return fmt.Errorf("redis not available")
	}

	ctx := context.Background()
	key := fmt.Sprintf("pending_host:%s", hostID)

	// 获取待准入主机信息
	data, err := redis.Get(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("pending host not found: %w", err)
	}

	var pendingHost models.PendingHost
	if err := json.Unmarshal([]byte(data), &pendingHost); err != nil {
		return fmt.Errorf("failed to unmarshal pending host: %w", err)
	}

	// 检查主机是否已经存在
	var existingHost models.Host
	result := hs.db.Where("host_id = ?", hostID).First(&existingHost)

	if result.Error == nil {
		// 主机已存在，更新状态为已准入
		existingHost.Status = models.HostStatusApproved
		if err := hs.db.Save(&existingHost).Error; err != nil {
			return fmt.Errorf("failed to update host status: %w", err)
		}
	} else if result.Error == gorm.ErrRecordNotFound {
		// 主机不存在，创建新记录
		host := pendingHost.ToHost()
		host.Status = models.HostStatusApproved
		if err := hs.db.Create(host).Error; err != nil {
			return fmt.Errorf("failed to create approved host: %w", err)
		}
	} else {
		return fmt.Errorf("failed to query existing host: %w", result.Error)
	}

	// 从待准入列表中删除
	if err := redis.Del(ctx, key).Err(); err != nil {
		// 记录错误但不返回，因为主机已经准入成功
		fmt.Printf("Warning: failed to remove pending host from Redis: %v\n", err)
	}

	return nil
}

// RejectHost 拒绝主机准入
func (hs *HostService) RejectHost(hostID string) error {
	redis := database.GetRedis()
	if redis == nil {
		return fmt.Errorf("redis not available")
	}

	ctx := context.Background()
	key := fmt.Sprintf("pending_host:%s", hostID)

	// 检查主机是否存在
	_, err := redis.Get(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("pending host not found: %w", err)
	}

	// 从待准入列表中删除（拒绝的主机直接删除）
	return redis.Del(ctx, key).Err()
}

// GetPendingHostsCount 获取待准入主机数量
func (hs *HostService) GetPendingHostsCount() (int, error) {
	redis := database.GetRedis()
	if redis == nil {
		return 0, fmt.Errorf("redis not available")
	}

	ctx := context.Background()
	keys, err := redis.Keys(ctx, "pending_host:*").Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get pending host keys: %w", err)
	}

	return len(keys), nil
}

// ReportHostStatus 处理主机状态上报
func (hs *HostService) ReportHostStatus(status *protobuf.HostStatus) error {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()

	// 检查主机是否已准入
	var host models.Host
	result := hs.db.Where("host_id = ? AND status = ?", status.HostId, models.HostStatusApproved).First(&host)

	if result.Error == gorm.ErrRecordNotFound {
		return fmt.Errorf("host not found or not approved: %s", status.HostId)
	} else if result.Error != nil {
		return fmt.Errorf("failed to query host: %w", result.Error)
	}

	// 更新主机最后上报时间
	host.LastSeen = time.Unix(status.Timestamp, 0)

	// 更新主机标签（合并状态信息）
	if host.Tags == nil {
		host.Tags = make(models.JSON)
	}

	// 添加系统状态信息到标签
	if status.Cpu != nil {
		host.Tags["cpu_usage"] = fmt.Sprintf("%.2f%%", status.Cpu.UsagePercent)
		host.Tags["cpu_cores"] = fmt.Sprintf("%d", status.Cpu.CoreCount)
	}

	if status.Memory != nil {
		host.Tags["memory_usage"] = fmt.Sprintf("%.2f%%", status.Memory.UsagePercent)
		host.Tags["memory_total"] = fmt.Sprintf("%.2fGB", float64(status.Memory.TotalBytes)/1024/1024/1024)
	}

	host.Tags["uptime"] = fmt.Sprintf("%ds", status.UptimeSeconds)

	// 更新 IP 地址
	if status.Ip != "" {
		host.IP = status.Ip
	}

	// 合并自定义标签
	for k, v := range status.CustomTags {
		host.Tags[k] = v
	}

	// 保存到数据库
	if err := hs.db.Save(&host).Error; err != nil {
		return fmt.Errorf("failed to update host status: %w", err)
	}

	// 缓存状态信息到 Redis
	hs.cacheHostStatus(status)

	return nil
}

// cacheHostStatus 缓存主机状态信息到 Redis
func (hs *HostService) cacheHostStatus(status *protobuf.HostStatus) {
	redis := database.GetRedis()
	if redis == nil {
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("host_status:%s", status.HostId)

	data, err := json.Marshal(status)
	if err != nil {
		return
	}

	// 缓存10分钟
	redis.Set(ctx, key, data, 10*time.Minute)
}

// GetHostStatus 获取主机状态信息
func (hs *HostService) GetHostStatus(hostID string) (*protobuf.HostStatus, error) {
	redis := database.GetRedis()
	if redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	ctx := context.Background()
	key := fmt.Sprintf("host_status:%s", hostID)

	data, err := redis.Get(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("host status not found: %w", err)
	}

	var status protobuf.HostStatus
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal host status: %w", err)
	}

	return &status, nil
}
