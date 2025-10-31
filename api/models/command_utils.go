package models

import (
	"crypto/rand"
	"devops-manager/api/protobuf"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// generateCommandID 生成命令ID
func generateCommandID() string {
	return fmt.Sprintf("cmd_%d_%d", time.Now().UnixNano(), randomInt())
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	return fmt.Sprintf("task_%d_%d", time.Now().UnixNano(), randomInt())
}

// randomInt 生成随机整数
func randomInt() int64 {
	b := make([]byte, 8)
	rand.Read(b)
	var result int64
	for i, v := range b {
		result |= int64(v) << (8 * i)
	}
	if result < 0 {
		result = -result
	}
	return result
}

// CommandBuilder 命令构建器
type CommandBuilder struct {
	command *Command
}

// NewCommandBuilder 创建新的命令构建器
func NewCommandBuilder() *CommandBuilder {
	return &CommandBuilder{
		command: &Command{
			CommandID:  generateCommandID(),
			Status:     CommandStatusPending,
			Parameters: "",
		},
	}
}

// WithTaskID 设置任务ID
func (cb *CommandBuilder) WithTaskID(taskID string) *CommandBuilder {
	cb.command.TaskID = &taskID
	return cb
}

// WithHostID 设置主机ID
func (cb *CommandBuilder) WithHostID(hostID string) *CommandBuilder {
	cb.command.HostID = hostID
	return cb
}

// WithCommand 设置命令内容
func (cb *CommandBuilder) WithCommand(cmd string) *CommandBuilder {
	cb.command.Command = cmd
	return cb
}

// WithParameter 设置命令参数（简单字符串格式）
func (cb *CommandBuilder) WithParameter(key, value string) *CommandBuilder {
	if cb.command.Parameters == "" {
		cb.command.Parameters = fmt.Sprintf("%s=%s", key, value)
	} else {
		cb.command.Parameters += fmt.Sprintf(" %s=%s", key, value)
	}
	return cb
}

// WithParameters 批量设置命令参数
func (cb *CommandBuilder) WithParameters(params map[string]string) *CommandBuilder {
	var paramPairs []string
	for k, v := range params {
		paramPairs = append(paramPairs, fmt.Sprintf("%s=%s", k, v))
	}
	if len(paramPairs) > 0 {
		if cb.command.Parameters == "" {
			cb.command.Parameters = fmt.Sprintf("%s", paramPairs[0])
			for i := 1; i < len(paramPairs); i++ {
				cb.command.Parameters += fmt.Sprintf(" %s", paramPairs[i])
			}
		} else {
			for _, pair := range paramPairs {
				cb.command.Parameters += fmt.Sprintf(" %s", pair)
			}
		}
	}
	return cb
}

// WithTimeout 设置超时时间（秒）
func (cb *CommandBuilder) WithTimeout(seconds int64) *CommandBuilder {
	cb.command.Timeout = seconds
	return cb
}

// Build 构建命令
func (cb *CommandBuilder) Build() *Command {
	cb.command.CreatedAt = time.Now()
	return cb.command
}

// CommandMessageHelper 命令消息辅助工具
type CommandMessageHelper struct{}

// NewCommandMessageHelper 创建命令消息辅助工具
func NewCommandMessageHelper() *CommandMessageHelper {
	return &CommandMessageHelper{}
}

// CreateCommandMessage 创建包含命令内容的消息
func (h *CommandMessageHelper) CreateCommandMessage(content *protobuf.CommandContent) *protobuf.CommandMessage {
	return &protobuf.CommandMessage{
		CommandContent: content,
	}
}

// CreateResultMessage 创建包含命令结果的消息
func (h *CommandMessageHelper) CreateResultMessage(result *protobuf.CommandResult) *protobuf.CommandMessage {
	return &protobuf.CommandMessage{
		CommandResult: result,
	}
}

// CommandFactory 命令工厂
type CommandFactory struct{}

// NewCommandFactory 创建命令工厂
func NewCommandFactory() *CommandFactory {
	return &CommandFactory{}
}

// CreateSimpleCommand 创建简单命令
func (f *CommandFactory) CreateSimpleCommand(hostID, command string) *protobuf.CommandContent {
	return &protobuf.CommandContent{
		CommandId:  generateCommandID(),
		HostId:     hostID,
		Command:    command,
		Parameters: "",                               // 现在是 string 类型
		Timeout:    durationpb.New(30 * time.Second), // 默认30秒超时
		CreatedAt:  timestamppb.Now(),
	}
}

// CreateCommandWithParams 创建带参数的命令
func (f *CommandFactory) CreateCommandWithParams(hostID, command string, params map[string]string, timeoutSeconds int64) *protobuf.CommandContent {
	// 将 map 转换为字符串格式
	var paramPairs []string
	for k, v := range params {
		paramPairs = append(paramPairs, fmt.Sprintf("%s=%s", k, v))
	}
	paramString := ""
	if len(paramPairs) > 0 {
		paramString = paramPairs[0]
		for i := 1; i < len(paramPairs); i++ {
			paramString += fmt.Sprintf(" %s", paramPairs[i])
		}
	}

	return &protobuf.CommandContent{
		CommandId:  generateCommandID(),
		HostId:     hostID,
		Command:    command,
		Parameters: paramString, // 现在是 string 类型
		Timeout:    durationpb.New(time.Duration(timeoutSeconds) * time.Second),
		CreatedAt:  timestamppb.Now(),
	}
}
