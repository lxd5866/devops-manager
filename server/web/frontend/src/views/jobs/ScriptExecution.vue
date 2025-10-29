<template>
  <div class="script-execution">
    <div class="page-header">
      <h1>脚本执行</h1>
      <p>创建新的脚本执行任务</p>
    </div>

    <a-card class="form-card">
      <a-form
        ref="formRef"
        :model="form"
        :rules="rules"
        :label-col="{ span: 4 }"
        :wrapper-col="{ span: 20 }"
        @finish="handleSubmit"
      >
        <a-form-item label="任务名称" name="name">
          <a-input
            v-model:value="form.name"
            placeholder="请输入任务名称"
            :maxlength="100"
            show-count
          />
        </a-form-item>

        <a-form-item label="目标主机" name="hostIds">
          <div class="host-selection">
            <a-button type="primary" @click="showHostModal = true">
              选择主机 ({{ selectedHosts.length }})
            </a-button>
            <div v-if="selectedHosts.length > 0" class="selected-hosts">
              <a-tag
                v-for="host in selectedHosts"
                :key="host.id"
                closable
                @close="removeHost(host.id)"
                class="host-tag"
              >
                {{ host.hostname }} ({{ host.ip }})
              </a-tag>
            </div>
          </div>
        </a-form-item>

        <a-form-item label="脚本内容" name="command">
          <a-textarea
            v-model:value="form.command"
            :rows="8"
            placeholder="请输入要执行的脚本内容"
            :maxlength="5000"
            show-count
          />
        </a-form-item>

        <a-form-item label="超时时间" name="timeout">
          <a-input-number
            v-model:value="form.timeout"
            :min="1"
            :max="3600"
            style="width: 200px"
          />
          <span class="input-suffix">秒</span>
        </a-form-item>

        <a-form-item label="脚本参数" name="parameters">
          <div class="parameters-section">
            <div
              v-for="(param, index) in form.parameters"
              :key="index"
              class="parameter-item"
            >
              <a-input
                v-model:value="param.key"
                placeholder="参数名"
                style="width: 200px; margin-right: 10px"
              />
              <a-input
                v-model:value="param.value"
                placeholder="参数值"
                style="width: 300px; margin-right: 10px"
              />
              <a-button
                type="primary"
                danger
                @click="removeParameter(index)"
                :disabled="form.parameters.length <= 1"
              >
                删除
              </a-button>
            </div>
            <a-button type="dashed" @click="addParameter">
              <template #icon><plus-outlined /></template>
              添加参数
            </a-button>
          </div>
        </a-form-item>

        <a-form-item label="任务描述" name="description">
          <a-textarea
            v-model:value="form.description"
            :rows="3"
            placeholder="请输入任务描述"
            :maxlength="500"
            show-count
          />
        </a-form-item>

        <a-form-item :wrapper-col="{ offset: 4, span: 20 }">
          <a-button type="primary" html-type="submit" :loading="submitting">
            提交执行
          </a-button>
          <a-button style="margin-left: 10px" @click="resetForm">重置</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 主机选择对话框 -->
    <a-modal
      v-model:open="showHostModal"
      title="选择目标主机"
      width="800px"
      @ok="confirmHostSelection"
      @cancel="handleHostModalClose"
    >
      <div class="host-modal-content">
        <a-input
          v-model:value="hostSearchKeyword"
          placeholder="搜索主机名或IP"
          style="margin-bottom: 20px"
        >
          <template #prefix><search-outlined /></template>
        </a-input>
        
        <a-table
          :data-source="filteredHosts"
          :row-selection="{ selectedRowKeys: tempSelectedHostIds, onChange: handleHostSelectionChange }"
          :pagination="false"
          :scroll="{ y: 400 }"
          size="small"
          row-key="id"
        >
          <a-table-column key="hostname" title="主机名" data-index="hostname" width="200" />
          <a-table-column key="ip" title="IP地址" data-index="ip" width="150" />
          <a-table-column key="os" title="操作系统" data-index="os" width="120" />
          <a-table-column key="status" title="状态" width="100">
            <template #default="{ record }">
              <a-tag color="green">
                已准入
              </a-tag>
            </template>
          </a-table-column>
        </a-table>
      </div>
      
      <template #footer>
        <a-button @click="showHostModal = false">取消</a-button>
        <a-button type="primary" @click="confirmHostSelection">
          确定 ({{ tempSelectedHostIds.length }})
        </a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { PlusOutlined, SearchOutlined } from '@ant-design/icons-vue'
import { hostsApi, tasksApi } from '@/api'

const router = useRouter()
const formRef = ref()

// 表单数据
const form = reactive({
  name: '',
  hostIds: [],
  command: '',
  timeout: 300,
  parameters: [{ key: '', value: '' }],
  description: ''
})

// 表单验证规则
const rules = {
  name: [
    { required: true, message: '请输入任务名称', trigger: 'blur' },
    { min: 1, max: 100, message: '任务名称长度在 1 到 100 个字符', trigger: 'blur' }
  ],
  hostIds: [
    { required: true, message: '请选择目标主机', trigger: 'change' }
  ],
  command: [
    { required: true, message: '请输入脚本内容', trigger: 'blur' },
    { min: 1, max: 5000, message: '脚本内容长度在 1 到 5000 个字符', trigger: 'blur' }
  ],
  timeout: [
    { required: true, message: '请输入超时时间', trigger: 'blur' },
    { type: 'number', min: 1, max: 3600, message: '超时时间在 1 到 3600 秒之间', trigger: 'blur' }
  ],
  description: [
    { required: true, message: '请输入任务描述', trigger: 'blur' },
    { min: 1, max: 500, message: '任务描述长度在 1 到 500 个字符', trigger: 'blur' }
  ]
}

// 状态变量
const submitting = ref(false)
const showHostModal = ref(false)
const hostSearchKeyword = ref('')
const hosts = ref([])
const selectedHosts = ref([])
const tempSelectedHostIds = ref([])

// 计算属性
const filteredHosts = computed(() => {
  // 确保hosts.value是数组
  const hostList = Array.isArray(hosts.value) ? hosts.value : []
  
  if (!hostSearchKeyword.value) return hostList
  
  const keyword = hostSearchKeyword.value.toLowerCase()
  return hostList.filter(host => 
    host.hostname?.toLowerCase().includes(keyword) ||
    host.ip?.toLowerCase().includes(keyword)
  )
})

// 生成默认任务名称
const generateTaskName = () => {
  const now = new Date()
  const timestamp = now.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).replace(/[\/\s:]/g, '')
  return `脚本执行-${timestamp}`
}

// 加载主机列表
const loadHosts = async () => {
  try {
    const response = await hostsApi.getHosts()
    // 确保response是数组
    hosts.value = Array.isArray(response) ? response : []
    console.log('Loaded hosts:', hosts.value)
  } catch (error) {
    message.error('加载主机列表失败: ' + (error.message || '未知错误'))
    console.error('Load hosts error:', error)
    // 确保在错误情况下hosts也是数组
    hosts.value = []
  }
}

// 主机选择相关方法
const handleHostSelectionChange = (selectedRowKeys) => {
  tempSelectedHostIds.value = selectedRowKeys
}

const confirmHostSelection = () => {
  selectedHosts.value = hosts.value.filter(host => 
    tempSelectedHostIds.value.includes(host.id)
  )
  form.hostIds = tempSelectedHostIds.value
  showHostModal.value = false
}

const handleHostModalClose = () => {
  tempSelectedHostIds.value = form.hostIds
}

const removeHost = (hostId) => {
  selectedHosts.value = selectedHosts.value.filter(host => host.id !== hostId)
  form.hostIds = selectedHosts.value.map(host => host.id)
}

// 参数管理方法
const addParameter = () => {
  form.parameters.push({ key: '', value: '' })
}

const removeParameter = (index) => {
  if (form.parameters.length > 1) {
    form.parameters.splice(index, 1)
  }
}

// 表单提交
const handleSubmit = async (values) => {
  try {
    submitting.value = true
    
    // 构建任务数据
    const taskData = {
      name: values.name,
      description: values.description,
      hostIds: values.hostIds,
      command: values.command,
      timeout: values.timeout,
      parameters: form.parameters.reduce((acc, param) => {
        if (param.key && param.value) {
          acc[param.key] = param.value
        }
        return acc
      }, {})
    }
    
    // 提交任务
    const response = await tasksApi.createTask(taskData)
    
    message.success('任务创建成功')
    
    // 跳转到执行历史页面
    router.push('/jobs/history')
    
  } catch (error) {
    message.error('任务创建失败: ' + (error.message || '未知错误'))
    console.error('Submit task error:', error)
  } finally {
    submitting.value = false
  }
}

// 重置表单
const resetForm = () => {
  formRef.value.resetFields()
  selectedHosts.value = []
  form.parameters = [{ key: '', value: '' }]
  form.name = generateTaskName()
}

// 初始化
onMounted(() => {
  form.name = generateTaskName()
  loadHosts()
})
</script>

<style scoped>
.script-execution {
  padding: 20px;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h1 {
  margin: 0 0 8px 0;
  font-size: 24px;
  color: #303133;
}

.page-header p {
  margin: 0;
  color: #909399;
}

.form-card {
  max-width: 800px;
}

.host-selection {
  width: 100%;
}

.selected-hosts {
  margin-top: 10px;
}

.host-tag {
  margin-right: 8px;
  margin-bottom: 8px;
}

.parameters-section {
  width: 100%;
}

.parameter-item {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}

.input-suffix {
  margin-left: 8px;
  color: #909399;
}

.host-dialog-content {
  max-height: 500px;
}
</style>