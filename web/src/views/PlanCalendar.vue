<template>
  <div>
    <div class="page-header">
      <h2>计划日历</h2>
      <button class="btn secondary" @click="triggerScan">触发调度器扫描</button>
    </div>
    <div v-if="error" class="error-message">{{ error }}</div>
    <div v-if="scanResult" class="success-message">
      扫描完成: 标记逾期 {{ scanResult.overdue_marked }} 项，生成待办 {{ scanResult.todos_generated }} 项，标记逾期待办 {{ scanResult.todos_overdue }} 项。
    </div>

    <div class="card">
      <h3>即将到期 / 已逾期计划</h3>
      <table v-if="plans.length">
        <thead>
          <tr>
            <th>设施</th><th>下次保养</th><th>距到期</th><th>状态</th><th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in plans" :key="p.id">
            <td>{{ p.facility_id }}</td>
            <td>{{ formatDate(p.next_due_date) }}</td>
            <td>{{ daysUntil(p.next_due_date) }} 天</td>
            <td><span class="tag" :class="planStatus(p)">{{ planStatus(p) === 'overdue' ? '已逾期' : '即将到期' }}</span></td>
            <td>
              <router-link :to="`/facilities/${p.facility_id}`">查看设施</router-link>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="muted">暂无计划</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { PlanService, SchedulerService } from '../api/services';
import type { MaintenancePlan } from '../types/models';

const plans = ref<MaintenancePlan[]>([]);
const error = ref('');
const scanResult = ref<any>(null);

function formatDate(s: string) {
  return new Date(s).toLocaleDateString('zh-CN');
}
function daysUntil(s: string) {
  const ms = new Date(s).getTime() - Date.now();
  return Math.ceil(ms / (1000 * 60 * 60 * 24));
}
function planStatus(p: MaintenancePlan) {
  return new Date(p.next_due_date).getTime() < Date.now() ? 'overdue' : 'pending_maintenance';
}

async function load() {
  error.value = '';
  try {
    const res = await PlanService.list({ limit: 50, order_by: 'next_due_date', order: 'asc' });
    plans.value = res.items || [];
  } catch (e: any) {
    error.value = e.message;
  }
}

async function triggerScan() {
  error.value = '';
  scanResult.value = null;
  try {
    scanResult.value = await SchedulerService.scan();
    await load();
  } catch (e: any) {
    error.value = e.message;
  }
}

onMounted(load);
</script>
