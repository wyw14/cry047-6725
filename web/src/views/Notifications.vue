<template>
  <div>
    <div class="page-header">
      <h2>提醒中心</h2>
    </div>
    <div v-if="error" class="error-message">{{ error }}</div>

    <div class="card">
      <table v-if="notifications.length">
        <thead>
          <tr>
            <th>类型</th><th>标题</th><th>正文</th><th>时间</th><th>已读</th><th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="n in notifications" :key="n.id">
            <td>{{ typeLabel(n.type) }}</td>
            <td>{{ n.title }}</td>
            <td>{{ n.body }}</td>
            <td>{{ formatDate(n.created_at) }}</td>
            <td>{{ n.read_at ? '是' : '否' }}</td>
            <td>
              <button v-if="!n.read_at" class="btn secondary" @click="markRead(n.id)">标记已读</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="muted">暂无通知</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { NotificationService } from '../api/services';
import { useAuthStore } from '../stores/auth';
import type { Notification } from '../types/models';

const auth = useAuthStore();
const notifications = ref<Notification[]>([]);
const error = ref('');

function typeLabel(t: string) {
  return {
    maintenance_due: '保养到期',
    maintenance_overdue: '保养逾期',
    anomaly_discovered: '异常发现',
    anomaly_recovered: '异常恢复',
    plan_updated: '计划更新',
    assigned: '待办分配',
  }[t] || t;
}
function formatDate(s: string) {
  return new Date(s).toLocaleString('zh-CN');
}

async function load() {
  error.value = '';
  try {
    const res = await NotificationService.listByUser(auth.actorId, { limit: 50 });
    notifications.value = res.items || [];
  } catch (e: any) {
    error.value = e.message;
  }
}

async function markRead(id: string) {
  try {
    await NotificationService.markRead(id);
    await load();
  } catch (e: any) {
    error.value = `[${e.code}] ${e.message}`;
  }
}

onMounted(load);
</script>
