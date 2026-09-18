import { localStg } from '@/utils/storage';

/** Get token */
export function getToken() {
  return localStg.get('token') || '';
}

/** Get the admin account info stored at login */
export function getAdminInfo(): Api.Auth.AdminUser | null {
  const str = localStg.get('adminInfo');

  if (!str) {
    return null;
  }

  try {
    return JSON.parse(str) as Api.Auth.AdminUser;
  } catch {
    return null;
  }
}

/** Clear auth storage */
export function clearAuthStorage() {
  localStg.remove('token');
  localStg.remove('refreshToken');
  localStg.remove('adminInfo');
}
