<template>
  <div class="app">
    <aside class="sidebar">
      <h1>保养协同平台</h1>
      <ul>
        <li><router-link to="/facilities" active-class="active">设施台账</router-link></li>
        <li><router-link to="/calendar" active-class="active">计划日历</router-link></li>
        <li><router-link to="/workbench" active-class="active">执行工作台</router-link></li>
        <li><router-link to="/anomalies" active-class="active">异常整改</router-link></li>
        <li><router-link to="/todos" active-class="active">个人待办</router-link></li>
        <li><router-link to="/notifications" active-class="active">提醒中心</router-link></li>
        <li><router-link to="/audit" active-class="active">审计日志</router-link></li>
        <li><router-link to="/config" active-class="active">配置页</router-link></li>
      </ul>
      <div class="muted" style="padding: 12px 20px; font-size: 12px; border-top: 1px solid #2c4053; margin-top: 16px;">
        当前角色: {{ auth.actorName }} ({{ auth.actorRole }})
        <div style="margin-top: 8px;">
          <select v-model="roleSel" @change="switchRole" style="font-size: 12px; padding: 2px 4px;">
            <option value="admin">管理员</option>
            <option value="supervisor">主管</option>
            <option value="engineer">工程师</option>
            <option value="operator">操作员</option>
            <option value="viewer">浏览者</option>
          </select>
        </div>
      </div>
    </aside>
    <main class="main">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { useAuthStore } from './stores/auth';

const auth = useAuthStore();
const roleSel = ref(auth.actorRole);

function switchRole() {
  const roleMap: Record<string, [string, string]> = {
    admin: ['admin-1', '管理员'],
    supervisor: ['sup-1', '主管'],
    engineer: ['eng-1', '工程师'],
    operator: ['op-1', '操作员'],
    viewer: ['viewer-1', '浏览者'],
  };
  const [id, name] = roleMap[roleSel.value] || ['viewer-1', '浏览者'];
  auth.set(id, name, roleSel.value as any);
}
</script>
