import { adminRequest } from '../request/admin';

/** GET /export/{kind} — download a CSV export of orders/users/photographers */
export async function exportAdminCsv(
  kind: 'orders' | 'users' | 'photographers',
  params?: Record<string, string>
) {
  const blob = await adminRequest.get<Blob, Blob>(`/export/${kind}`, { params, responseType: 'blob' });

  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `${kind}.csv`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

/** GET /dashboard — overview totals, trends and order status distribution */
export function fetchAdminDashboard() {
  return adminRequest.get<Api.Admin.Dashboard, Api.Admin.Dashboard>('/dashboard');
}

/** GET /photographers — list all photographers, optional certified filter */
export async function fetchAdminPhotographers(certified?: boolean) {
  const data = await adminRequest.get<Api.Admin.Photographer[] | null, Api.Admin.Photographer[] | null>(
    '/photographers',
    {
      params: certified === undefined ? undefined : { certified: String(certified) }
    }
  );

  // the Go backend returns a null body (nil slice) when the filtered list is empty
  return data ?? [];
}

/** GET /photographers/:id — photographer detail with stats */
export function fetchAdminPhotographerDetail(id: number) {
  return adminRequest.get<Api.Admin.PhotographerDetail, Api.Admin.PhotographerDetail>(`/photographers/${id}`);
}

/** GET /audit-logs — paginated admin operation log */
export async function fetchAuditLogs(params: { page?: number; pageSize?: number }) {
  const data = await adminRequest.get<Api.Admin.AuditLogList, Api.Admin.AuditLogList>('/audit-logs', { params });
  return data ?? { list: [], total: 0 };
}

/** PUT /photographers/:id/certified — pass or revoke the certification */
export function updateAdminPhotographerCertified(id: number, certified: boolean) {
  return adminRequest.put<{ ok: boolean }, { ok: boolean }>(`/photographers/${id}/certified`, { certified });
}

/** GET /orders — paginated order list, optional status filter */
export async function fetchAdminOrders(params: { status?: string; page?: number; pageSize?: number }) {
  const data = await adminRequest.get<Api.Admin.OrderList | null, Api.Admin.OrderList | null>('/orders', { params });

  return data ?? { list: [], total: 0 };
}

/** PUT /orders/:id/status — update order status following the state machine */
export function updateAdminOrderStatus(id: number, status: Api.Admin.OrderStatus) {
  return adminRequest.put<{ ok: boolean }, { ok: boolean }>(`/orders/${id}/status`, { status });
}

/** GET /events — all comic events (may be 100+ rows, filter client-side) */
export async function fetchAdminEvents() {
  const data = await adminRequest.get<Api.Admin.EventItem[] | null, Api.Admin.EventItem[] | null>('/events');

  // the Go backend returns a null body (nil slice) when there are no events
  return data ?? [];
}

/** PUT /events/:id/status — put on/off shelf via delFlag */
export function updateAdminEventStatus(id: number, delFlag: boolean) {
  return adminRequest.put<{ ok: boolean }, { ok: boolean }>(`/events/${id}/status`, { delFlag });
}

/** GET /users — paginated user list, optional keyword/role/status filters */
export async function fetchAdminUsers(params: {
  keyword?: string;
  role?: string;
  status?: string;
  page?: number;
  pageSize?: number;
}) {
  const data = await adminRequest.get<Api.Admin.UserList | null, Api.Admin.UserList | null>('/users', { params });

  return data ?? { list: [], total: 0 };
}

/** GET /users/:id — user profile with stats and recent bookings */
export function fetchAdminUserDetail(id: number) {
  return adminRequest.get<Api.Admin.AdminUserDetail, Api.Admin.AdminUserDetail>(`/users/${id}`);
}

/** PUT /users/:id/status — enable or disable a user */
export function setUserStatus(id: number, status: Api.Admin.UserStatus) {
  return adminRequest.put<{ ok: boolean }, { ok: boolean }>(`/users/${id}/status`, { status });
}

/** GET /cert-applications — paginated certification applications, optional status filter */
export async function fetchCertApplications(params: { status?: string; page?: number; pageSize?: number }) {
  const data = await adminRequest.get<Api.Admin.CertApplicationList | null, Api.Admin.CertApplicationList | null>(
    '/cert-applications',
    { params }
  );

  return data ?? { list: [], total: 0 };
}

/** GET /cert-applications/:id — single certification application */
export function fetchCertApplicationDetail(id: number) {
  return adminRequest.get<Api.Admin.AdminCertApplication, Api.Admin.AdminCertApplication>(
    `/cert-applications/${id}`
  );
}

/** PUT /cert-applications/:id/review — approve or reject (reason required on reject) */
export function reviewCertApplication(id: number, body: { action: 'approve' | 'reject'; reason?: string }) {
  return adminRequest.put<{ ok: boolean }, { ok: boolean }>(`/cert-applications/${id}/review`, body);
}

/** GET /banners — full banner list sorted by sortOrder asc */
export async function fetchBanners() {
  const data = await adminRequest.get<Api.Admin.AdminBanner[] | null, Api.Admin.AdminBanner[] | null>('/banners');

  return data ?? [];
}

/** POST /banners — create a banner */
export function createBanner(payload: Partial<Api.Admin.AdminBanner>) {
  return adminRequest.post<{ id: number }, { id: number }>('/banners', payload);
}

/** PUT /banners/:id — update banner fields (always include isActive) */
export function updateBanner(id: number, payload: Partial<Api.Admin.AdminBanner>) {
  return adminRequest.put<{ ok: boolean }, { ok: boolean }>(`/banners/${id}`, payload);
}

/** PUT /banners/:id/status — enable or disable a banner */
export function setBannerStatus(id: number, isActive: boolean) {
  return adminRequest.put<{ ok: boolean }, { ok: boolean }>(`/banners/${id}/status`, { isActive });
}

/** GET /admins — list all admin accounts */
export async function fetchAdmins() {
  const data = await adminRequest.get<Api.Admin.AdminAccount[] | null, Api.Admin.AdminAccount[] | null>('/admins');

  return data ?? [];
}

/** POST /admins — create an admin account */
export function createAdmin(payload: { username: string; password: string; role: Api.Admin.AdminRole }) {
  return adminRequest.post<{ id: number }, { id: number }>('/admins', payload);
}

/** PUT /admins/:id/status — enable or disable an admin account */
export function setAdminStatus(id: number, status: Api.Admin.AdminStatus) {
  return adminRequest.put<{ ok: boolean }, { ok: boolean }>(`/admins/${id}/status`, { status });
}

/** PUT /admins/:id/password — reset an admin password (invalidates the target's token) */
export function resetAdminPassword(id: number, password: string) {
  return adminRequest.put<{ ok: boolean }, { ok: boolean }>(`/admins/${id}/password`, { password });
}

/** POST /notifications — broadcast a notification to all users or a single user */
export function publishNotification(payload: Api.Admin.PublishNotificationRequest) {
  return adminRequest.post<{ id: number }, { id: number }>('/notifications', payload);
}

/** DELETE /notifications/:id — recall a published notification */
export function deleteNotification(id: number) {
  return adminRequest.delete<{ ok: boolean }, { ok: boolean }>(`/notifications/${id}`);
}

/** GET /notifications — paginated notification broadcast history */
export async function fetchNotifications(params: { page?: number; pageSize?: number }) {
  const data = await adminRequest.get<Api.Admin.NotificationList | null, Api.Admin.NotificationList | null>(
    '/notifications',
    { params }
  );

  return data ?? { list: [], total: 0 };
}

/** GET /works — paginated work list, optional photographerId/status filters */
export async function fetchAdminWorks(params: {
  photographerId?: number;
  status?: string;
  page?: number;
  pageSize?: number;
}) {
  const data = await adminRequest.get<Api.Admin.AdminWorkList | null, Api.Admin.AdminWorkList | null>('/works', {
    params
  });

  return data ?? { list: [], total: 0 };
}

/** PUT /works/:id/status — put a work on/off shelf ('active' | 'down') */
export function setWorkStatus(id: number, status: 'active' | 'down') {
  return adminRequest.put<{ ok: boolean }, { ok: boolean }>(`/works/${id}/status`, { status });
}

/** GET /reviews — paginated review list, optional keyword/photographerId filters */
export async function fetchReviews(params: {
  keyword?: string;
  photographerId?: number;
  page?: number;
  pageSize?: number;
}) {
  const data = await adminRequest.get<Api.Admin.AdminReviewList | null, Api.Admin.AdminReviewList | null>(
    '/reviews',
    { params }
  );

  return data ?? { list: [], total: 0 };
}

/** DELETE /reviews/:id — permanently remove a review */
export function deleteReview(id: number) {
  return adminRequest.delete<null, null>(`/reviews/${id}`);
}

/** GET /tags — all style tags with real usage counts */
export async function fetchTags() {
  const data = await adminRequest.get<Api.Admin.AdminTag[] | null, Api.Admin.AdminTag[] | null>('/tags');

  return data ?? [];
}

/** POST /tags — create a style tag */
export function createTag(payload: { name: string }) {
  return adminRequest.post<{ id: number }, { id: number }>('/tags', payload);
}

/** PUT /tags/:id — rename a style tag */
export function updateTag(id: number, payload: { name: string }) {
  return adminRequest.put<{ ok: boolean }, { ok: boolean }>(`/tags/${id}`, payload);
}

/** DELETE /tags/:id — remove a style tag (400 when still in use) */
export function deleteTag(id: number) {
  return adminRequest.delete<null, null>(`/tags/${id}`);
}

/** POST /tags/merge — merge a tag into another, moving its references */
export function mergeTags(fromId: number, toId: number) {
  return adminRequest.post<{ ok: boolean }, { ok: boolean }>('/tags/merge', { fromId, toId });
}
