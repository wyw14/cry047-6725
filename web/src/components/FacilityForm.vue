<template>
  <form @submit.prevent="onSubmit">
    <div class="mb-8">
      <label>场所 *</label>
      <select v-model="form.place_id" required>
        <option value="">请选择</option>
        <option v-for="p in places" :key="p.id" :value="p.id">{{ p.name }}</option>
      </select>
    </div>
    <div class="mb-8">
      <label>名称 *</label>
      <input v-model="form.name" required maxlength="128" />
    </div>
    <div class="mb-8">
      <label>编码 *（3-32 位大写字母数字或短横线）</label>
      <input v-model="form.code" required pattern="[A-Z0-9-]{3,32}" />
    </div>
    <div class="mb-8">
      <label>类别 *</label>
      <input v-model="form.category" required />
    </div>
    <div class="mb-8">
      <label>责任人 *</label>
      <select v-model="form.responsible_person_id" required>
        <option value="">请选择</option>
        <option v-for="p in people" :key="p.id" :value="p.id">{{ p.name }} ({{ p.department }})</option>
      </select>
    </div>
    <div class="mb-8">
      <label>关键等级 *</label>
      <select v-model="form.criticality" required>
        <option value="critical">关键</option>
        <option value="important">重要</option>
        <option value="standard">标准</option>
      </select>
    </div>
    <div class="mb-8">
      <label>描述</label>
      <textarea v-model="form.description" maxlength="1024"></textarea>
    </div>
    <div class="flex">
      <button type="submit" class="btn">保存</button>
      <button type="button" class="btn secondary" @click="$emit('cancel')">取消</button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { reactive } from 'vue';
import type { Place, ResponsiblePerson } from '../types/models';

const props = defineProps<{
  places: Place[];
  people: ResponsiblePerson[];
}>();

const emit = defineEmits<{ submit: [any]; cancel: [] }>();

const form = reactive({
  place_id: '',
  name: '',
  code: '',
  category: '',
  responsible_person_id: '',
  criticality: 'standard',
  description: '',
});

function onSubmit() {
  emit('submit', { ...form });
}
</script>
