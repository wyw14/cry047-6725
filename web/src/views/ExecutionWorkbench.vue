<template>
  <div>
    <div class="page-header">
      <h2>执行工作台</h2>
      <button class="btn" @click="showForm = !showForm">{{ showForm ? '收起' : '提交保养记录' }}</button>
    </div>
    <div v-if="error" class="error-message">{{ error }}</div>

    <div v-if="showForm" class="card">
      <h3>提交保养执行记录</h3>
      <div class="mb-8">
        <label>计划 ID</label>
        <input v-model="form.plan_id" required />
      </div>
      <div class="mb-8">
        <label>设施 ID</label>
        <input v-model="form.facility_id" required />
      </div>
      <div class="mb-8">
        <label>幂等键 *（用于防止重复提交）</label>
        <input v-model="form.idempotency_key" required />
      </div>
      <div class="mb-8">
        <label>执行时间</label>
        <input type="datetime-local" v-model="form.executed_at" />
      </div>
      <div class="mb-8">
        <label>检查项 (JSON)</label>
        <textarea v-model="form.inspection_values_json" rows="4" placeholder='[{"item_code":"filter","pass":true}]'></textarea>
      </div>
      <div class="flex">
        <button class="btn" @click="submit">提交</button>
      </div>
    </div>

    <div class="card">
      <h3>最近执行记录</h3>
      <table v-if="executions.length">
        <thead>
          <tr>
            <th>设施</th><th>执行人</th><th>执行时间</th><th>状态</th><th>复核</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in executions" :key="e.id">
            <td>{{ e.facility_id }}</td>
            <td>{{ e.executed_by }}</td>
            <td>{{ formatDate(e.executed_at) }}</td>
            <td><span class="tag">{{ statusLabel(e.status) }}</span></td>
            <td>
              <button v-if="e.status === 'submitted'" class="btn secondary" @click="review(e.id, true)">通过</button>
              <button v-if="e.status === 'submitted'" class="btn danger" @click="review(e.id, false)">驳回</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="muted">暂无执行记录</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { ExecutionService } from '../api/services';
import type { Execution } from '../types/models';

const executions = ref<Execution[]>([]);
const error = ref('');
const showForm = ref(false);

const form = reactive({
  plan_id: '',
  facility_id: '',
  idempotency_key: '',
  executed_at: '',
  inspection_values_json: '[]',
});

function formatDate(s: string) {
  return new Date(s).toLocaleString('zh-CN');
}
function statusLabel(s: string) {
  return { draft: '草稿', submitted: '已提交', reviewed: '已复核', rejected: '已驳回' }[s] || s;
}

async function load() {
  error.value = '';
  try {
    const res = await ExecutionService.list({ limit: 20, order_by: 'executed_at', order: 'desc' });
    executions.value = res.items || [];
  } catch (e: any) {
    error.value = e.message;
  }
}

async function submit() {
  error.value = '';
  try {
    let values: any = JSON.parse(form.inspection_values_json || '[]');
    await ExecutionService.submit({
      plan_id: form.plan_id,
      facility_id: form.facility_id,
      idempotency_key: form.idempotency_key,
      executed_at: form.executed_at ? new Date(form.executed_at).toISOString() : undefined,
      inspection_values: values,
      consumables_consumed: [],
      photos: [],
    });
    await load();
    showForm.value = false;
    form.plan_id = '';
    form.facility_id = '';
    form.idempotency_key = '';
  } catch (e: any) {
    error.value = `[${e.code}] ${e.message}`;
  }
}

async function review(id: string, approved: boolean) {
  const comment = prompt(`请输入复核意见（${approved ? '通过' : '驳回'}）`);
  if (comment === null) return;
  try {
    await ExecutionService.review(id, comment, approved);
    await load();
  } catch (e: any) {
    error.value = `[${e.code}] ${e.message}`;
  }
}

onMounted(load);
</script>
