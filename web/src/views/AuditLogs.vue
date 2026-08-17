<template>
  <div>
    <div class="page-header">
      <h2>审计日志</h2>
      <button class="btn secondary" @click="load">刷新</button>
    </div>
    <div v-if="error" class="error-message">{{ error }}</div>

    <div class="card">
      <table v-if="logs.length">
        <thead>
          <tr>
            <th>时间</th><th>操作</th><th>实体</th><th>实体 ID</th><th>操作人</th><th>原因</th><th>request_id</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="l in logs" :key="l.id">
            <td>{{ formatDate(l.created_at) }}</td>
            <td>{{ l.action }}</td>
            <td>{{ l.entity_type }}</td>
            <td>{{ l.entity_id }}</td>
            <td>{{ l.actor_name }}</td>
            <td>{{ l.reason }}</td>
            <td><code>{{ l.request_id }}</code></td>
          </tr>
        </tbody>
      </table>
      <div v-else class="muted">暂无审计日志</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { AuditService } from '../api/services';
import type { AuditLog } from '../types/models';

const logs = ref<AuditLog[]>([]);
const error = ref('');

function formatDate(s: string) {
  return new Date(s).toLocaleString('zh-CN');
}

async function load() {
  error.value = '';
  try {
    const res = await AuditService.list({ limit: 100 });
    logs.value = res.items || [];
  } catch (e: any) {
    error.value = e.message;
  }
}

onMounted(load);
</script>
