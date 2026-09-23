<template>
  <div>
    <div class="page-header">
      <h3>API 配置</h3>
      <a-button type="primary" @click="openCreate">
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
              <div v-if="record.hasDraft && record.draft && record.draft.method !== record.method" class="draft-diff">
                <a-tag size="small" color="orangered">{{ record.draft.method }}</a-tag>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="路径" data-index="path">
            <template #cell="{ record }">
              <code>{{ record.path }}</code>
              <div v-if="record.hasDraft && record.draft && record.draft.path !== record.path" class="draft-diff">
                <code class="draft-text">{{ record.draft.path }}</code>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="状态码" data-index="statusCode" width="100">
            <template #cell="{ record }">
              <a-tag>{{ record.statusCode }}</a-tag>
              <div v-if="record.hasDraft && record.draft && record.draft.statusCode !== record.statusCode" class="draft-diff">
                <a-tag size="small" color="orangered">{{ record.draft.statusCode }}</a-tag>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="发布状态" width="130">
            <template #cell="{ record }">
              <a-tooltip v-if="record.hasDraft" :mini="false">
                <a-tag color="orangered" icon>
                  <template #icon><icon-exclamation-circle-fill /></template>
                  未发布
                </a-tag>
                <template #content>
                  <div>该接口有未发布草稿，外部请求仍按线上版本响应：</div>
                  <div v-for="item in draftChangeSummary(record)" :key="item">· {{ item }}</div>
                  <div v-if="record.draft">草稿保存于 {{ formatTime(record.draft.updatedAt) }}</div>
                </template>
              </a-tooltip>
              <a-tag v-else color="green">已上线</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="延迟" data-index="delay" width="80">
            <template #cell="{ record }">
              {{ record.delay || 0 }}ms
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
          <a-table-column title="操作" width="230">
            <template #cell="{ record }">
              <a-space wrap>
                <a-button type="text" size="small" @click="openEdit(record)">
                  编辑
                </a-button>
                <a-button
                  v-if="record.hasDraft"
                  type="text"
                  size="small"
                  status="success"
                  @click="handleQuickPublish(record)"
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
      v-model:visible="showModal"
      :title="editingAPI ? '编辑 API' : '新建 API'"
      :width="800"
      :mask-closable="false"
      @cancel="handleCancel"
    >
      <a-alert v-if="editingAPI && editingAPI.hasDraft" type="warning" class="draft-alert">
        当前编辑的是未发布草稿，外部请求仍按线上版本响应；确认无误后请点击「发布」，改坏了可「放弃草稿」回到线上版本。
      </a-alert>
      <a-alert v-else-if="editingAPI" type="info" class="draft-alert">
        修改后请点击「存草稿」或「发布」；仅存草稿不会影响当前线上响应。
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
        <a-collapse>
          <a-collapse-panel header="响应头配置">
            <div class="headers-section">
              <div v-for="(header, index) in apiForm.responseHeaders" :key="index" class="header-item">
                <a-input v-model="header.key" placeholder="Header 名称，如 X-Demo" class="header-key" />
                <a-input v-model="header.value" placeholder="Header 值" class="header-value" />
                <a-button type="text" status="danger" @click="removeHeader(index)">
                  <icon-delete />
                </a-button>
              </div>
              <a-button type="outline" size="small" @click="addHeader">
                <template #icon><icon-plus /></template>
                添加响应头
              </a-button>
            </div>
          </a-collapse-panel>
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
        <a-space>
          <template v-if="editingAPI">
            <a-button
              status="warning"
              :disabled="!editingAPI.hasDraft"
              @click="handleDiscard"
            >
              放弃草稿
            </a-button>
            <a-button @click="handleSaveDraft">存草稿</a-button>
            <a-button type="primary" status="success" @click="handlePublish">发布</a-button>
          </template>
          <template v-else>
            <a-button @click="showModal = false">取消</a-button>
            <a-button type="primary" @click="handleCreate">创建</a-button>
          </template>
        </a-space>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { Message, Modal } from '@arco-design/web-vue';
import {
  IconPlus,
  IconCopy,
  IconDelete,
  IconExclamationCircleFill
} from '@arco-design/web-vue/es/icon';
import { useProjectStore } from '../store';
import type { MockAPI, ConditionRule } from '../types';
import type { EndpointPayload } from '../api';
import MonacoEditor from '../components/MonacoEditor.vue';

interface HeaderPair {
  key: string;
  value: string;
}

const route = useRoute();
const projectStore = useProjectStore();

const showModal = ref(false);
const editingAPI = ref<MockAPI | null>(null);
const apiForm = ref({
  method: 'GET',
  path: '',
  statusCode: 200,
  responseBody: '{}',
  delay: 0,
  responseHeaders: [] as HeaderPair[],
  conditions: [] as ConditionRule[]
});

const projectId = computed(() => route.params.projectId as string);
const draftCount = computed(() => projectStore.apis.filter((a) => a.hasDraft).length);

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

function formatTime(value: string) {
  if (!value) return '';
  return value.replace('T', ' ').slice(0, 19);
}

function headersToPairs(headers?: Record<string, string>): HeaderPair[] {
  return Object.entries(headers ?? {}).map(([key, value]) => ({ key, value }));
}

function resetForm() {
  editingAPI.value = null;
  apiForm.value = {
    method: 'GET',
    path: '',
    statusCode: 200,
    responseBody: '{}',
    delay: 0,
    responseHeaders: [],
    conditions: []
  };
}

function fillForm(api: MockAPI, useDraft: boolean) {
  const source = useDraft && api.draft ? api.draft : api;
  editingAPI.value = api;
  apiForm.value = {
    method: source.method,
    path: source.path,
    statusCode: source.statusCode,
    responseBody: source.responseBody ?? '{}',
    delay: source.delay ?? 0,
    responseHeaders: headersToPairs(source.responseHeaders),
    conditions: (source.conditions ?? []).map((c) => ({ ...c }))
  };
}

function openCreate() {
  resetForm();
  showModal.value = true;
}

function openEdit(api: MockAPI) {
  if (api.hasDraft && api.draft) {
    Modal.confirm({
      title: '检测到未发布草稿',
      content: '该接口存在尚未发布的草稿，将打开草稿内容继续编辑；线上响应不受影响。',
      okText: '编辑草稿',
      cancelText: '查看线上版本',
      onOk: () => {
        fillForm(api, true);
        showModal.value = true;
      },
      onCancel: () => {
        fillForm(api, false);
        showModal.value = true;
      }
    });
    return;
  }
  fillForm(api, false);
  showModal.value = true;
}

function buildPayload(): EndpointPayload | null {
  const path = apiForm.value.path.trim();
  if (!path.startsWith('/')) {
    Message.warning('API 路径必须以 / 开头');
    return null;
  }
  const headers: Record<string, string> = {};
  for (const h of apiForm.value.responseHeaders) {
    const key = h.key.trim();
    if (key) headers[key] = h.value;
  }
  return {
    path,
    method: apiForm.value.method,
    statusCode: apiForm.value.statusCode,
    responseBody: apiForm.value.responseBody,
    responseHeaders: headers,
    delay: apiForm.value.delay,
    conditions: apiForm.value.conditions
  };
}

function errorMessage(error: any, fallback: string) {
  return error?.response?.data?.message || error?.response?.data?.error || fallback;
}

function refreshEditing(api: MockAPI | undefined) {
  if (api) editingAPI.value = api;
}

async function handleCreate() {
  const payload = buildPayload();
  if (!payload) return;
  try {
    const result = await projectStore.createAPI(projectId.value, payload);
    if (result.success) {
      Message.success('创建成功，已上线');
      showModal.value = false;
      resetForm();
    }
  } catch (error: any) {
    Message.error(errorMessage(error, '创建失败'));
  }
}

async function handleSaveDraft() {
  if (!editingAPI.value) return;
  const payload = buildPayload();
  if (!payload) return;
  try {
    const result = await projectStore.saveAPIDraft(projectId.value, editingAPI.value._id, payload);
    if (result.success) {
      refreshEditing(result.data);
      Message.success('草稿已保存，线上响应保持不变');
    }
  } catch (error: any) {
    Message.error(errorMessage(error, '草稿保存失败'));
  }
}

async function publish(id: string, payload?: EndpointPayload) {
  const result = await projectStore.publishAPIDraft(projectId.value, id, payload);
  if (result.success) {
    Message.success('发布成功，新配置已接管线上响应');
    showModal.value = false;
    resetForm();
  }
}

function handlePublish() {
  if (!editingAPI.value) return;
  const payload = buildPayload();
  if (!payload) return;
  const id = editingAPI.value._id;
  Modal.confirm({
    title: '确认发布',
    content: '发布后新配置将立即接管线上 Mock 响应，外部请求将按当前草稿内容返回。确认发布？',
    okText: '发布',
    onOk: async () => {
      try {
        await publish(id, payload);
      } catch (error: any) {
        Message.error(errorMessage(error, '发布失败'));
      }
    }
  });
}

function handleQuickPublish(api: MockAPI) {
  Modal.confirm({
    title: '确认发布',
    content: `将发布「${api.method} ${api.path}」的未发布草稿，新配置立即接管线上响应。`,
    okText: '发布',
    onOk: async () => {
      try {
        await publish(api._id);
      } catch (error: any) {
        Message.error(errorMessage(error, '发布失败'));
      }
    }
  });
}

function handleDiscard() {
  if (!editingAPI.value || !editingAPI.value.hasDraft) return;
  const id = editingAPI.value._id;
  Modal.confirm({
    title: '放弃草稿',
    content: '放弃后编辑内容将恢复为当前线上版本，未发布的草稿无法恢复。确认放弃？',
    okText: '放弃草稿',
    status: 'warning',
    onOk: async () => {
      try {
        const result = await projectStore.discardAPIDraft(projectId.value, id);
        if (result.success) {
          Message.success('已放弃草稿，恢复为线上版本');
          showModal.value = false;
          resetForm();
        }
      } catch (error: any) {
        Message.error(errorMessage(error, '放弃草稿失败'));
      }
    }
  });
}

function handleCancel() {
  showModal.value = false;
  resetForm();
}

function handleDelete(api: MockAPI) {
  const others = draftCount.value - (api.hasDraft ? 1 : 0);
  const lines: string[] = [];
  if (api.hasDraft) {
    lines.push('该接口存在未发布草稿，删除会同时丢弃草稿与线上配置，请确认已完成联调。');
  }
  if (others > 0) {
    lines.push(`当前项目还有 ${others} 个接口存在未发布草稿，请勿误删联调中的配置。`);
  }
  Modal.confirm({
    title: '确认删除',
    content: [
      `确定要删除 API「${api.method} ${api.path}」吗？删除后不可恢复。`,
      ...lines
    ].join('\n'),
    okText: api.hasDraft ? '确认强制删除' : '删除',
    status: 'danger',
    onOk: async () => {
      try {
        const result = await projectStore.deleteAPI(projectId.value, api._id, api.hasDraft);
        if (result.success) {
          Message.success('删除成功');
          if (editingAPI.value?._id === api._id) {
            showModal.value = false;
            resetForm();
          }
        }
      } catch (error: any) {
        Message.error(errorMessage(error, '删除失败'));
      }
    }
  });
}

function draftChangeSummary(api: MockAPI): string[] {
  if (!api.draft) return [];
  const changes: string[] = [];
  if (api.draft.path !== api.path) changes.push(`路径：${api.path} → ${api.draft.path}`);
  if (api.draft.method !== api.method) changes.push(`方法：${api.method} → ${api.draft.method}`);
  if (api.draft.statusCode !== api.statusCode) {
    changes.push(`状态码：${api.statusCode} → ${api.draft.statusCode}`);
  }
  const headerCount = Object.keys(api.draft.responseHeaders ?? {}).length;
  const liveHeaderCount = Object.keys(api.responseHeaders ?? {}).length;
  if (headerCount !== liveHeaderCount || api.draft.delay !== api.delay) {
    changes.push(`响应头/延迟：${liveHeaderCount} 个头、${api.delay}ms → ${headerCount} 个头、${api.draft.delay}ms`);
  }
  if (api.draft.responseBody !== api.responseBody) changes.push('响应体已修改');
  if ((api.draft.conditions?.length ?? 0) !== (api.conditions?.length ?? 0)) {
    changes.push(`条件规则：${api.conditions?.length ?? 0} 条 → ${api.draft.conditions?.length ?? 0} 条`);
  }
  return changes.length ? changes : ['草稿内容与线上一致（可直接发布或放弃）'];
}

function addHeader() {
  apiForm.value.responseHeaders.push({ key: '', value: '' });
}

function removeHeader(index: number) {
  apiForm.value.responseHeaders.splice(index, 1);
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

.draft-alert {
  margin-bottom: 16px;
}

.draft-diff {
  margin-top: 4px;
}

.draft-text {
  color: rgb(var(--orange-6));
  font-size: 12px;
}

.headers-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.header-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-key {
  width: 240px;
  flex-shrink: 0;
}

.header-value {
  flex: 1;
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
