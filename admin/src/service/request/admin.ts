import axios from 'axios';
import { localStg } from '@/utils/storage';
import { clearAuthStorage } from '@/store/modules/auth/shared';

/**
 * Admin backend base url, points to the Go backend (mira comic admin API)
 */
export const adminBaseURL = import.meta.env.VITE_ADMIN_API_BASE_URL || 'http://localhost:8080/api/admin/v1';

/** Dedicated axios instance for the mira admin API (plain REST JSON, no envelope) */
export const adminRequest = axios.create({
  baseURL: adminBaseURL,
  timeout: 15000
});

adminRequest.interceptors.request.use(config => {
  const token = localStg.get('token');

  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }

  return config;
});

adminRequest.interceptors.response.use(
  response => response.data,
  error => {
    const status = error.response?.status;
    const url = String(error.config?.url || '');
    const isLoginRequest = url.includes('/login');

    // 401 means the token is invalid/expired, clear it and go back to the login page
    // the login request itself is excluded to avoid a redirect loop
    if (status === 401 && !isLoginRequest) {
      clearAuthStorage();
      window.location.replace('/login');
    }

    return Promise.reject(error);
  }
);

/** Extract a readable message from an admin api error */
export function getAdminApiErrorMessage(error: unknown): string {
  const err = error as { response?: { data?: { error?: string; message?: string } }; message?: string };

  return err.response?.data?.error || err.response?.data?.message || err.message || '请求失败';
}
