<template>
  <div>
    <div class="page-header">
      <h2>个人待办</h2>
      <div class="flex">
        <input v-model="userId" placeholder="用户 ID" />
        <button class="btn" @click="load">查询</button>
      </div>
    </div>
    <div v-if="error" class="error-message">{{ error }}</div>

    <div class="card">
      <table v-if="todos.length">
        <thead>
          <tr>
            <th>标题</th><th>类型</th><th>截止日期</th><th>状态</th><th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in todos" :key="t.id">
            <td>{{ t.title }}</td>
            <td>{{ typeLabel(t.type) }}</td>
            <td>{{ formatDate(t.due_date) }}</td>
            <td><span class="tag" :class="t.status">{{ statusLabel(t.status) }}</span></td>
            <td>
              <button v-if="t.status !== 'completed'" class="btn" @click="complete(t.id)">完成</button>
              <button v-if="t.status !== 'completed'" class="btn secondary" @click="reassign(t.id)">转派</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="muted">暂无待办</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { TodoService } from '../api/services';
import type { Todo } from '../types/models';

const todos = ref<Todo[]>([]);
const userId = ref('admin-1');
const error = ref('');

function typeLabel(t: string) {
  return { maintenance: '保养', rectification: '整改', reinspection: '复检', recovery: '恢复', review: '复核' }[t] || t;
}
function statusLabel(s: string) {
  return { open: '待办', assigned: '已分配', completed: '已完成', skipped: '已跳过', overdue: '逾期' }[s] || s;
}
function formatDate(s: string) {
  return new Date(s).toLocaleString('zh-CN');
}

async function load() {
  error.value = '';
  try {
    const res = await TodoService.listByUser(userId.value, { limit: 50 });
    todos.value = res.items || [];
  } catch (e: any) {
    error.value = e.message;
  }
}

async function complete(id: string) {
  try {
    await TodoService.complete(id);
    await load();
  } catch (e: any) {
    error.value = `[${e.code}] ${e.message}`;
  }
}

async function reassign(id: string) {
  const personId = prompt('转派到（责任人 ID）');
  if (personId === null) return;
  const reason = prompt('转派原因') || '';
  try {
    await TodoService.reassign(id, personId, reason);
    await load();
  } catch (e: any) {
    error.value = `[${e.code}] ${e.message}`;
  }
}

onMounted(load);
</script>
