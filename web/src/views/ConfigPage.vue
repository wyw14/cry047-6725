<template>
  <div>
    <div class="page-header">
      <h2>配置页</h2>
    </div>
    <div v-if="error" class="error-message">{{ error }}</div>

    <div class="card">
      <h3>场所管理</h3>
      <table v-if="places.length">
        <thead><tr><th>编码</th><th>名称</th><th>类型</th><th>地址</th></tr></thead>
        <tbody>
          <tr v-for="p in places" :key="p.id">
            <td>{{ p.code }}</td>
            <td>{{ p.name }}</td>
            <td>{{ p.type }}</td>
            <td>{{ p.address }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="card">
      <h3>责任人管理</h3>
      <table v-if="people.length">
        <thead><tr><th>姓名</th><th>部门</th><th>邮箱</th><th>电话</th><th>状态</th></tr></thead>
        <tbody>
          <tr v-for="p in people" :key="p.id">
            <td>{{ p.name }}</td>
            <td>{{ p.department }}</td>
            <td>{{ p.email }}</td>
            <td>{{ p.phone }}</td>
            <td>{{ p.active ? '在职' : '停用' }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="card">
      <h3>计划模板管理</h3>
      <table v-if="templates.length">
        <thead><tr><th>名称</th><th>周期天数</th><th>检查项</th><th>耗材</th><th>需停用</th></tr></thead>
        <tbody>
          <tr v-for="t in templates" :key="t.id">
            <td>{{ t.name }}</td>
            <td>{{ t.cycle_days }}</td>
            <td>{{ t.inspection_items.length }}</td>
            <td>{{ t.consumables.length }}</td>
            <td>{{ t.requires_shutdown ? '是' : '否' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { PlaceService, PersonService, TemplateService } from '../api/services';
import type { Place, ResponsiblePerson, PlanTemplate } from '../types/models';

const places = ref<Place[]>([]);
const people = ref<ResponsiblePerson[]>([]);
const templates = ref<PlanTemplate[]>([]);
const error = ref('');

async function load() {
  error.value = '';
  try {
    const [p, rp, t] = await Promise.all([
      PlaceService.list({ limit: 100 }),
      PersonService.list({ limit: 100 }),
      TemplateService.list({ limit: 100 }),
    ]);
    places.value = p.items || [];
    people.value = rp.items || [];
    templates.value = t.items || [];
  } catch (e: any) {
    error.value = e.message;
  }
}

onMounted(load);
</script>
