import { describe, it, expect, vi, beforeEach } from 'vitest';
import { apiClient } from '../api/client';

describe('api client', () => {
  beforeEach(() => {
    apiClient.defaults.baseURL = '/api/v1';
  });

  it('has the correct base URL', () => {
    expect(apiClient.defaults.baseURL).toBe('/api/v1');
  });

  it('sets default actor headers', () => {
    expect(apiClient.defaults.headers.common['X-Actor-Role']).toBeTruthy();
  });
});
