import { describe, it, expect, beforeEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useAuthStore } from '../stores/auth';

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it('initializes with default actor', () => {
    const auth = useAuthStore();
    expect(auth.actorId).toBe('admin-1');
    expect(auth.actorRole).toBe('admin');
  });

  it('updates the actor on set()', () => {
    const auth = useAuthStore();
    auth.set('op-1', '操作员', 'operator');
    expect(auth.actorId).toBe('op-1');
    expect(auth.actorName).toBe('操作员');
    expect(auth.actorRole).toBe('operator');
  });
});
