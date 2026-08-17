<template>
  <div>
    <div class="page-header">
      <h2>设施详情</h2>
      <router-link to="/facilities">返回台账</router-link>
    </div>

    <div v-if="error" class="error-message">{{ error }}</div>

    <div v-if="facility" class="card">
      <h3>{{ facility.name }} <span class="muted">({{ facility.code }})</span></h3>
      <p><strong>类别:</strong> {{ facility.category }}</p>
      <p><strong>关键等级:</strong> <span class="tag" :class="facility.criticality">{{ criticalityLabel(facility.criticality) }}</span></p>
      <p><strong>状态:</strong> <span class="tag" :class="facility.status">{{ statusLabel(facility.status) }}</span></p>
      <p><strong>描述:</strong> {{ facility.description || '—' }}</p>

      <h4 class="mt-16">下次保养</h4>
      <div v-if="nextInfo">
        <p>下次保养日期: {{ formatDate(nextInfo.next_due_date) }}</p>
        <p>距到期: {{ nextInfo.days_until_due }} 天</p>
        <p>逾期风险: <span class="tag" :class="nextInfo.overdue_risk ? 'overdue' : 'normal'">{{ nextInfo.overdue_risk ? '是' : '否' }}</span></p>
        <p v-if="nextInfo.alternative_facility_id">替代设施 ID: {{ nextInfo.alternative_facility_id }}</p>
      </div>
    </div>

    <div class="card">
      <h3>状态转换</h3>
      <div class="flex" style="gap: 8px; flex-wrap: wrap;">
        <button v-for="s in allowedTransitions" :key="s" class="btn" @click="transition(s)">{{ statusLabel(s) }}</button>
      </div>
      <div v-if="transitionMsg" class="error-message mt-8">{{ transitionMsg }}</div>
    </div>

    <div class="card">
      <h3>时间线</h3>
      <div v-if="timeline.length">
        <div v-for="t in timeline" :key="t.id" class="timeline-item">
          <div class="time">{{ formatDate(t.occurred_at) }}</div>
          <div class="title">{{ t.title }}</div>
          <div class="actor">操作人: {{ t.actor }}</div>
        </div>
      </div>
      <div v-else class="muted">暂无时间线事件</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue';
import { useRoute } from 'vue-router';
import { FacilityService } from '../api/services';
import type { Facility, NextDateInfo, TimelineEvent, FacilityStatus } from '../types/models';

const route = useRoute();
const facility = ref<Facility | null>(null);
const nextInfo = ref<NextDateInfo | null>(null);
const timeline = ref<TimelineEvent[]>([]);
const error = ref('');
const transitionMsg = ref('');

function criticalityLabel(c: string) {
  return { critical: '关键', important: '重要', standard: '标准' }[c] || c;
}
function statusLabel(s: string) {
  return {
    normal: '正常',
    pending_maintenance: '待保养',
    overdue: '逾期',
    restricted_use: '限用',
    under_repair: '维修中',
    recovered: '恢复',
  }[s] || s;
}

function formatDate(s?: string) {
  if (!s) return '—';
  return new Date(s).toLocaleString('zh-CN');
}

// Simplified allowed transitions for UI display; backend enforces the real
// state machine (including the "overdue critical cannot be normal" rule).
const allowedTransitions: FacilityStatus[] = [
  'normal', 'pending_maintenance', 'overdue', 'restricted_use', 'under_repair', 'recovered',
];

async function load() {
  const id = route.params.id as string;
  error.value = '';
  try {
    facility.value = await FacilityService.get(id);
    nextInfo.value = await FacilityService.nextDate(id);
    timeline.value = await FacilityService.timeline(id, 50);
  } catch (e: any) {
    error.value = e.message;
  }
}

async function transition(to: FacilityStatus) {
  if (!facility.value) return;
  transitionMsg.value = '';
  const reason = prompt(`请输入状态变更原因 (${statusLabel(facility.value.status)} → ${statusLabel(to)})`);
  if (reason === null) return;
  try {
    facility.value = await FacilityService.transition(facility.value.id, to, reason || '');
    timeline.value = await FacilityService.timeline(facility.value.id, 50);
  } catch (e: any) {
    transitionMsg.value = `[${e.code}] ${e.message}`;
  }
}

onMounted(load);
watch(() => route.params.id, load);
</script>
