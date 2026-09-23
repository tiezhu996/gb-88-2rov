<template>
  <div>
    <div class="page-header">
      <h3>API 配置</h3>
      <a-button type="primary" @click="showCreateModal = true">
        <template #icon><icon-plus /></template>
        新建 API
      </a-button>
    </div>

    <a-card :loading="projectStore.loading">
      <a-table :data="projectStore.apis" :pagination="false">
        <template #columns>
          <a-table-column title="方法" data-index="method" width="100">
            <template #cell="{ record }">
              <a-tag :color="getMethodColor(record.method)">{{ record.method }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="路径" data-index="path">
            <template #cell="{ record }">
              <code>{{ record.path }}</code>
            </template>
          </a-table-column>
          <a-table-column title="状态码" data-index="statusCode" width="100">
            <template #cell="{ record }">
              <a-tag>{{ record.statusCode }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="延迟" data-index="delay" width="80">
            <template #cell="{ record }">
              {{ record.delay || 0 }}ms
            </template>
          </a-table-column>
          <a-table-column title="发布状态" width="100">
            <template #cell="{ record }">
              <a-tag v-if="record.hasDraft" color="orange">未发布</a-tag>
              <a-tag v-else color="green">已发布</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="Mock URL" width="250">
            <template #cell="{ record }">
              <div class="url-container">
                <code class="mock-url">{{ getMockUrl(record) }}</code>
                <a-button type="text" size="mini" @click="copyUrl(getMockUrl(record))">
                  <icon-copy />
                </a-button>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="操作" width="200">
            <template #cell="{ record }">
              <a-space>
                <a-button type="text" size="small" @click="handleEdit(record)">
                  编辑
                </a-button>
                <a-button
                  v-if="record.hasDraft"
                  type="text"
                  size="small"
                  status="success"
                  @click="handleRowPublish(record)"
                >
                  发布
                </a-button>
                <a-button type="text" size="small" status="danger" @click="handleDelete(record)">
                  删除
                </a-button>
              </a-space>
            </template>
          </a-table-column>
        </template>
        <template #empty>
          <a-empty description="暂无 API，点击右上角新建 API" />
        </template>
      </a-table>
    </a-card>

    <a-modal
      v-model:visible="showCreateModal"
      :title="modalTitle"
      :width="800"
      @close="resetForm"
    >
      <a-alert v-if="editingAPI?.hasDraft" type="warning" style="margin-bottom: 16px">
        正在编辑未发布的草稿（保存于 {{ formatTime(editingAPI.draftSavedAt) }}）。
        线上请求仍按上一版配置响应，点击「发布」后新配置才会接管线上。
      </a-alert>
      <a-form :model="apiForm" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item field="method" label="请求方法">
              <a-select v-model="apiForm.method" style="width: 100%">
                <a-option value="GET">GET</a-option>
                <a-option value="POST">POST</a-option>
                <a-option value="PUT">PUT</a-option>
                <a-option value="DELETE">DELETE</a-option>
                <a-option value="PATCH">PATCH</a-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item field="statusCode" label="响应状态码">
              <a-input-number v-model="apiForm.statusCode" :min="100" :max="599" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item field="path" label="API 路径">
          <a-input v-model="apiForm.path" placeholder="/api/users/:id" />
        </a-form-item>
        <a-form-item field="delay" label="响应延迟 (毫秒)">
          <a-input-number v-model="apiForm.delay" :min="0" style="width: 200px" />
        </a-form-item>
        <a-form-item field="responseBody" label="响应体 (JSON)">
          <MonacoEditor v-model="apiForm.responseBody" language="json" />
        </a-form-item>
        <a-form-item field="responseHeaders" label="响应头">
          <div class="headers-section">
            <div v-for="(header, index) in apiForm.responseHeaders" :key="index" class="header-item">
              <a-input v-model="header.key" placeholder="Header 名，如 X-Env" />
              <a-input v-model="header.value" placeholder="值" />
              <a-button type="text" status="danger" @click="apiForm.responseHeaders.splice(index, 1)">
                <icon-delete />
              </a-button>
            </div>
            <a-button type="outline" size="small" @click="apiForm.responseHeaders.push({ key: '', value: '' })">
              <template #icon><icon-plus /></template>
              添加响应头
            </a-button>
          </div>
        </a-form-item>
        <a-collapse>
          <a-collapse-panel header="条件响应配置">
            <div class="conditions-section">
              <a-button type="outline" size="small" @click="addCondition">
                <template #icon><icon-plus /></template>
                添加条件
              </a-button>
              <div v-for="(condition, index) in apiForm.conditions" :key="index" class="condition-item">
                <a-row :gutter="8" align="middle">
                  <a-col :span="5">
                    <a-input v-model="condition.field" placeholder="字段名" />
                  </a-col>
                  <a-col :span="4">
                    <a-select v-model="condition.operator" style="width: 100%">
                      <a-option value="equals">等于</a-option>
                      <a-option value="contains">包含</a-option>
                      <a-option value="startsWith">开头</a-option>
                      <a-option value="endsWith">结尾</a-option>
                    </a-select>
                  </a-col>
                  <a-col :span="5">
                    <a-input v-model="condition.value" placeholder="匹配值" />
                  </a-col>
                  <a-col :span="3">
                    <a-input-number v-model="condition.statusCode" :min="100" :max="599" style="width: 100%" />
                  </a-col>
                  <a-col :span="6">
                    <a-input v-model="condition.responseBody" placeholder="响应体" />
                  </a-col>
                  <a-col :span="1">
                    <a-button type="text" status="danger" @click="removeCondition(index)">
                      <icon-delete />
                    </a-button>
                  </a-col>
                </a-row>
              </div>
            </div>
          </a-collapse-panel>
        </a-collapse>
      </a-form>
      <template #footer>
        <a-button @click="showCreateModal = false">取消</a-button>
        <a-button
          v-if="editingAPI?.hasDraft"
          status="danger"
          :loading="discarding"
          @click="handleDiscard"
        >
          放弃草稿
        </a-button>
        <a-button
          v-if="editingAPI"
          type="primary"
          :loading="publishing"
          @click="handlePublish"
        >
          发布
        </a-button>
        <a-button
          :type="editingAPI ? 'outline' : 'primary'"
          :loading="saving"
          @click="handleSaveDraft"
        >
          {{ editingAPI ? '保存草稿' : '创建' }}
        </a-button>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { Message, Modal } from '@arco-design/web-vue';
import { IconPlus, IconCopy, IconDelete } from '@arco-design/web-vue/es/icon';
import { useProjectStore } from '../store';
import type { MockAPI } from '../types';
import MonacoEditor from '../components/MonacoEditor.vue';

interface HeaderItem {
  key: string;
  value: string;
}

const route = useRoute();
const projectStore = useProjectStore();

const showCreateModal = ref(false);
const editingAPI = ref<MockAPI | null>(null);
const saving = ref(false);
const publishing = ref(false);
const discarding = ref(false);
const apiForm = ref({
  method: 'GET',
  path: '',
  statusCode: 200,
  responseBody: '{}',
  responseHeaders: [] as HeaderItem[],
  delay: 0,
  conditions: [] as any[]
});

const projectId = computed(() => route.params.projectId as string);
const modalTitle = computed(() => {
  if (!editingAPI.value) return '新建 API';
  return editingAPI.value.hasDraft ? '编辑 API（草稿）' : '编辑 API';
});

function getMethodColor(method: string) {
  const colors: Record<string, string> = {
    GET: 'green',
    POST: 'blue',
    PUT: 'orange',
    DELETE: 'red',
    PATCH: 'purple'
  };
  return colors[method] || 'gray';
}

function getMockUrl(api: MockAPI) {
  return `/mock/${projectId.value}${api.path}`;
}

function copyUrl(url: string) {
  navigator.clipboard.writeText(window.location.origin + url);
  Message.success('已复制');
}

function formatTime(t?: string) {
  return t ? new Date(t).toLocaleString() : '';
}

function headersToList(headers?: Record<string, string>): HeaderItem[] {
  return Object.entries(headers || {}).map(([key, value]) => ({ key, value }));
}

function headersToMap(list: HeaderItem[]): Record<string, string> {
  const map: Record<string, string> = {};
  for (const h of list) {
    const key = h.key.trim();
    if (key) {
      map[key] = h.value;
    }
  }
  return map;
}

function handleEdit(api: MockAPI) {
  editingAPI.value = api;
  // Resume the unpublished draft when one exists; otherwise edit the live config.
  apiForm.value = api.hasDraft
    ? {
        method: api.draftMethod || api.method,
        path: api.draftPath || api.path,
        statusCode: api.draftStatusCode || api.statusCode,
        responseBody: api.draftResponseBody ?? '',
        responseHeaders: headersToList(api.draftResponseHeaders),
        delay: api.draftDelay ?? api.delay,
        conditions: [...(api.draftConditions || [])]
      }
    : {
        method: api.method,
        path: api.path,
        statusCode: api.statusCode,
        responseBody: api.responseBody,
        responseHeaders: headersToList(api.responseHeaders),
        delay: api.delay,
        conditions: [...api.conditions]
      };
  showCreateModal.value = true;
}

function buildPayload() {
  return {
    method: apiForm.value.method,
    path: apiForm.value.path,
    statusCode: apiForm.value.statusCode,
    responseBody: apiForm.value.responseBody,
    responseHeaders: headersToMap(apiForm.value.responseHeaders),
    delay: apiForm.value.delay,
    conditions: apiForm.value.conditions
  };
}

function validateForm() {
  if (!apiForm.value.path) {
    Message.warning('请输入 API 路径');
    return false;
  }
  return true;
}

async function handleSaveDraft() {
  if (!validateForm()) return;
  saving.value = true;
  try {
    if (editingAPI.value) {
      const result = await projectStore.updateAPI(projectId.value, editingAPI.value._id, buildPayload());
      if (result.success) {
        Message.success('草稿已保存，发布后才会生效');
      }
    } else {
      const result = await projectStore.createAPI(projectId.value, buildPayload());
      if (result.success) {
        Message.success('创建成功');
      }
    }
    showCreateModal.value = false;
  } catch (error: any) {
    Message.error(error.response?.data?.error || '保存失败');
  } finally {
    saving.value = false;
  }
}

async function handlePublish() {
  if (!editingAPI.value || !validateForm()) return;
  publishing.value = true;
  try {
    // Persist the current edits as the draft first, then promote it.
    const saved = await projectStore.updateAPI(projectId.value, editingAPI.value._id, buildPayload());
    if (!saved.success) return;
    if (!saved.data?.hasDraft) {
      Message.info('没有需要发布的变更');
      showCreateModal.value = false;
      return;
    }
    const result = await projectStore.publishAPI(projectId.value, editingAPI.value._id);
    if (result.success) {
      Message.success('已发布，新配置已接管线上响应');
      showCreateModal.value = false;
    }
  } catch (error: any) {
    Message.error(error.response?.data?.error || '发布失败');
  } finally {
    publishing.value = false;
  }
}

function handleRowPublish(api: MockAPI) {
  Modal.confirm({
    title: '发布草稿',
    content: `确定将「${api.method} ${api.path}」的未发布草稿发布为线上版本吗？发布后新配置立即接管线上响应。`,
    onOk: async () => {
      try {
        const result = await projectStore.publishAPI(projectId.value, api._id);
        if (result.success) {
          Message.success('已发布，新配置已接管线上响应');
        }
      } catch (error: any) {
        Message.error(error.response?.data?.error || '发布失败');
      }
    }
  });
}

function handleDiscard() {
  if (!editingAPI.value) return;
  const target = editingAPI.value;
  Modal.confirm({
    title: '放弃草稿',
    content: '放弃后将回到当前线上版本，未发布的修改会丢失。确定放弃吗？',
    okButtonProps: { status: 'danger' },
    onOk: async () => {
      discarding.value = true;
      try {
        const result = await projectStore.discardDraft(projectId.value, target._id);
        if (result.success) {
          Message.success('已放弃草稿，恢复为线上版本');
          showCreateModal.value = false;
        }
      } catch (error: any) {
        Message.error(error.response?.data?.error || '操作失败');
      } finally {
        discarding.value = false;
      }
    }
  });
}

function handleDelete(api: MockAPI) {
  // Warn before cleaning up an endpoint that still has an unpublished draft.
  if (api.hasDraft) {
    confirmForceDelete(api);
    return;
  }
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除 API「${api.method} ${api.path}」吗？`,
    onOk: async () => {
      try {
        const result = await projectStore.deleteAPI(projectId.value, api._id);
        if (result.success) {
          Message.success('删除成功');
        }
      } catch (error: any) {
        if (error.response?.data?.code === 40900) {
          // A draft appeared after the list was loaded; ask again explicitly.
          confirmForceDelete(api);
        } else {
          Message.error(error.response?.data?.error || '删除失败');
        }
      }
    }
  });
}

function confirmForceDelete(api: MockAPI) {
  Modal.warning({
    title: '该接口存在未发布的草稿',
    content: `「${api.method} ${api.path}」还有未发布的草稿修改，删除后草稿将一并丢弃，线上配置也会立即停止服务。确定继续删除吗？`,
    okText: '仍然删除',
    okButtonProps: { status: 'danger' },
    onOk: async () => {
      try {
        const result = await projectStore.deleteAPI(projectId.value, api._id, true);
        if (result.success) {
          Message.success('已删除（含未发布草稿）');
        }
      } catch (error: any) {
        Message.error(error.response?.data?.error || '删除失败');
      }
    }
  });
}

function addCondition() {
  apiForm.value.conditions.push({
    field: '',
    operator: 'equals',
    value: '',
    responseBody: '{}',
    statusCode: 200
  });
}

function removeCondition(index: number) {
  apiForm.value.conditions.splice(index, 1);
}

function resetForm() {
  editingAPI.value = null;
  apiForm.value = {
    method: 'GET',
    path: '',
    statusCode: 200,
    responseBody: '{}',
    responseHeaders: [],
    delay: 0,
    conditions: []
  };
}

onMounted(() => {
  projectStore.fetchAPIs(projectId.value);
});
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h3 {
  margin: 0;
}

.url-container {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mock-url {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.headers-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.header-item {
  display: grid;
  grid-template-columns: 1fr 1fr 32px;
  gap: 8px;
  align-items: center;
}

.conditions-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.condition-item {
  padding: 12px;
  background: #f7f8fa;
  border-radius: 4px;
}
</style>
