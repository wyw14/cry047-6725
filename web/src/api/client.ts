import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';
import type { ApiResponse, ApiError } from '../types/models';

// Default actor used for demo navigation. In a real deployment this would
// come from a JWT or session cookie.
const DEFAULT_ACTOR = {
  'X-Actor-Id': 'admin-1',
  'X-Actor-Name': '管理员',
  'X-Actor-Role': 'admin',
};

// apiClient is the singleton axios instance used by every service.
export const apiClient: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 10_000,
});

// Apply default actor headers.
apiClient.defaults.headers.common = { ...apiClient.defaults.headers.common, ...DEFAULT_ACTOR };

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response) {
      const body = error.response.data as ApiError;
      return Promise.reject({
        status: error.response.status,
        code: body?.code ?? 'INTERNAL',
        message: body?.message ?? '请求失败',
        errors: body?.errors,
        request_id: body?.request_id,
      } as ApiError);
    }
    return Promise.reject({
      code: 'INTERNAL',
      message: error.message || '网络错误',
    } as ApiError);
  },
);

// get performs a GET request and returns the data field of the response.
export async function get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
  const res = await apiClient.get<ApiResponse<T>>(url, config);
  return res.data.data as T;
}

// post performs a POST request and returns the data field of the response.
export async function post<T>(url: string, body?: any, config?: AxiosRequestConfig): Promise<T> {
  const res = await apiClient.post<ApiResponse<T>>(url, body, config);
  return res.data.data as T;
}

// patch performs a PATCH request and returns the data field of the response.
export async function patch<T>(url: string, body?: any, config?: AxiosRequestConfig): Promise<T> {
  const res = await apiClient.patch<ApiResponse<T>>(url, body, config);
  return res.data.data as T;
}

// del performs a DELETE request and returns void.
export async function del(url: string, config?: AxiosRequestConfig): Promise<void> {
  await apiClient.delete(url, config);
}

// setActor updates the actor headers used by the apiClient.
export function setActor(id: string, name: string, role: string) {
  apiClient.defaults.headers.common['X-Actor-Id'] = id;
  apiClient.defaults.headers.common['X-Actor-Name'] = encodeURIComponent(name);
  apiClient.defaults.headers.common['X-Actor-Role'] = role;
}
