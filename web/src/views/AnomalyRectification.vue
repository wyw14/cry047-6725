<template>
  <div>
    <div class="page-header">
      <h2>异常整改</h2>
    </div>
    <div v-if="error" class="error-message">{{ error }}</div>

    <div class="card">
      <table v-if="anomalies.length">
        <thead>
          <tr>
            <th>设施</th><th>严重度</th><th>状态</th><th>描述</th><th>发现时间</th><th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in anomalies" :key="a.id">
            <td>{{ a.facility_id }}</td>
            <td><span class="tag" :class="a.severity">{{ severityLabel(a.severity) }}</span></td>
            <td><span class="tag">{{ statusLabel(a.status) }}</span></td>
            <td>{{ a.description }}</td>
            <td>{{ formatDate(a.discovered_at) }}</td>
            <td>
              <button v-if="a.status === 'open'" class="btn" @click="rectify(a.id)">整改</button>
              <button v-if="a.status === 'reinspecting'" class="btn secondary" @click="reinspect(a.id, true)">复检通过</button>
              <button v-if="a.status === 'reinspecting'" class="btn danger" @click="reinspect(a.id, false)">复检驳回</button>
              <button v-if="a.status === 'recovered'" class="btn success" @click="recover(a.id)">恢复确认</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="muted">暂无异常</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { AnomalyService } from '../api/services';
import type { Anomaly } from '../types/models';

const anomalies = ref<Anomaly[]>([]);
const error = ref('');

function severityLabel(s: string) {
  return { low: '低', medium: '中', high: '高', critical: '紧急' }[s] || s;
}
function statusLabel(s: string) {
  return { open: '待整改', rectifying: '整改中', reinspecting: '待复检', recovered: '已恢复待确认', closed_no_action: '关闭' }[s] || s;
}
function formatDate(s: string) {
  return new Date(s).toLocaleString('zh-CN');
}

async function load() {
  error.value = '';
  try {
    const res = await AnomalyService.list({ limit: 50, order_by: 'discovered_at', order: 'desc' });
    anomalies.value = res.items || [];
  } catch (e: any) {
    error.value = e.message;
  }
}

async function rectify(id: string) {
  const measure = prompt('请输入整改措施');
  if (measure === null) return;
  try {
    await AnomalyService.rectify(id, measure);
    await load();
  } catch (e: any) {
    error.value = `[${e.code}] ${e.message}`;
  }
}

async function reinspect(id: string, pass: boolean) {
  const result = prompt(`请输入复检结果（${pass ? '通过' : '驳回'}）`);
  if (result === null) return;
  try {
    await AnomalyService.reinspect(id, result, pass);
    await load();
  } catch (e: any) {
    error.value = `[${e.code}] ${e.message}`;
  }
}

async function recover(id: string) {
  if (!confirm('确认恢复？')) return;
  try {
    await AnomalyService.recover(id);
    await load();
  } catch (e: any) {
    error.value = `[${e.code}] ${e.message}`;
  }
}

onMounted(load);
</script>
