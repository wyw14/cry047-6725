<template>
  <div>
    <div class="page-header">
      <h2>设施台账</h2>
      <button class="btn" @click="showCreate = true">新建设施</button>
    </div>

    <div class="card">
      <div class="flex mb-8" style="gap: 12px; flex-wrap: wrap;">
        <input v-model="filters.name" placeholder="按名称搜索" />
        <select v-model="filters.criticality">
          <option value="">全部关键等级</option>
          <option value="critical">关键</option>
          <option value="important">重要</option>
          <option value="standard">标准</option>
        </select>
        <select v-model="filters.status">
          <option value="">全部状态</option>
          <option value="normal">正常</option>
          <option value="pending_maintenance">待保养</option>
          <option value="overdue">逾期</option>
          <option value="restricted_use">限用</option>
          <option value="under_repair">维修中</option>
          <option value="recovered">恢复</option>
        </select>
        <select v-model="filters.place_id" v-if="places.length">
          <option value="">全部场所</option>
          <option v-for="p in places" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
        <button class="btn secondary" @click="resetFilters">重置</button>
      </div>

      <div v-if="error" class="error-message">{{ error }}</div>

      <table v-if="facilities.length">
        <thead>
          <tr>
            <th>编码</th><th>名称</th><th>类别</th><th>关键等级</th><th>状态</th><th>负责人</th><th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="f in facilities" :key="f.id">
            <td>{{ f.code }}</td>
            <td>{{ f.name }}</td>
            <td>{{ f.category }}</td>
            <td><span class="tag" :class="f.criticality">{{ criticalityLabel(f.criticality) }}</span></td>
            <td><span class="tag" :class="f.status">{{ statusLabel(f.status) }}</span></td>
            <td>{{ personName(f.responsible_person_id) }}</td>
            <td>
              <router-link :to="`/facilities/${f.id}`">详情</router-link>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="muted">暂无设施数据</div>

      <div class="flex-between mt-16" v-if="total > limit">
        <span class="muted">共 {{ total }} 条</span>
        <div class="flex">
          <button class="btn secondary" :disabled="offset === 0" @click="prev">上一页</button>
          <button class="btn secondary" :disabled="offset + limit >= total" @click="next">下一页</button>
        </div>
      </div>
    </div>

    <div v-if="showCreate" class="card">
      <h3>新建设施</h3>
      <FacilityForm :places="places" :people="people" @submit="onCreate" @cancel="showCreate = false" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue';
import { FacilityService, PlaceService, PersonService } from '../api/services';
import type { Facility, Place, ResponsiblePerson } from '../types/models';
import FacilityForm from '../components/FacilityForm.vue';

const facilities = ref<Facility[]>([]);
const places = ref<Place[]>([]);
const people = ref<ResponsiblePerson[]>([]);
const total = ref(0);
const limit = ref(20);
const offset = ref(0);
const error = ref('');
const showCreate = ref(false);

const filters = reactive({
  name: '',
  criticality: '',
  status: '',
  place_id: '',
});

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
function personName(id: string) {
  return people.value.find((p) => p.id === id)?.name || id;
}

async function load() {
  error.value = '';
  try {
    const params: any = { limit: limit.value, offset: offset.value };
    if (filters.name) params.name = filters.name;
    if (filters.criticality) params.criticality = filters.criticality;
    if (filters.status) params.status = filters.status;
    if (filters.place_id) params.place_id = filters.place_id;
    const res = await FacilityService.list(params);
    facilities.value = res.items || [];
    total.value = res.total;
  } catch (e: any) {
    error.value = e.message;
  }
}

function resetFilters() {
  filters.name = '';
  filters.criticality = '';
  filters.status = '';
  filters.place_id = '';
  offset.value = 0;
  load();
}

function prev() {
  offset.value = Math.max(0, offset.value - limit.value);
  load();
}
function next() {
  offset.value += limit.value;
  load();
}

async function onCreate(payload: any) {
  try {
    await FacilityService.create(payload);
    showCreate.value = false;
    await load();
  } catch (e: any) {
    error.value = e.message;
  }
}

onMounted(async () => {
  await load();
  try {
    const [p, rp] = await Promise.all([
      PlaceService.list({ limit: 100 }),
      PersonService.list({ limit: 100 }),
    ]);
    places.value = p.items || [];
    people.value = rp.items || [];
  } catch (e: any) {
    error.value = e.message;
  }
});

watch(filters, () => { offset.value = 0; load(); }, { deep: true });
</script>
