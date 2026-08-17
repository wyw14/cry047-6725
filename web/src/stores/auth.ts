import { defineStore } from 'pinia';
import { ref } from 'vue';
import { setActor } from '../api/client';
import type { Role } from '../types/models';

// useAuthStore is a tiny Pinia store that manages the current actor identity.
// In a real deployment this would be backed by JWT validation; the platform is
// intentionally offline, so we manage it client-side.
export const useAuthStore = defineStore('auth', () => {
  const actorId = ref('admin-1');
  const actorName = ref('管理员');
  const actorRole = ref<Role>('admin');

  function set(id: string, name: string, role: Role) {
    actorId.value = id;
    actorName.value = name;
    actorRole.value = role;
    setActor(id, name, role);
  }

  // Initialize with default values.
  set(actorId.value, actorName.value, actorRole.value);

  return { actorId, actorName, actorRole, set };
});
