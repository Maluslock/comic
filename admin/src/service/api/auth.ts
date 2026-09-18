import { adminRequest } from '../request/admin';

/**
 * Admin login
 *
 * @param userName User name
 * @param password Password
 */
export function fetchLogin(userName: string, password: string) {
  return adminRequest.post<Api.Auth.LoginToken, Api.Auth.LoginToken>('/login', {
    username: userName,
    password
  });
}
