declare namespace Api {
  /**
   * namespace Admin
   *
   * DTOs of the mira comic admin API (Go backend :8080)
   */
  namespace Admin {
    interface DashboardTotals {
      users: number;
      photographers: number;
      certified: number;
      orders: number;
      pendingOrders: number;
      events: number;
    }

    interface DashboardTrends {
      date: string[];
      newUsers: number[];
      activeUsers: number[];
      orders: number[];
    }

    interface OrdersByStatus {
      pending: number;
      confirmed: number;
      completed: number;
      cancelled: number;
    }

    interface Dashboard {
      totals: DashboardTotals;
      trends: DashboardTrends;
      ordersByStatus: OrdersByStatus;
    }

    interface Photographer {
      id: number;
      name: string;
      avatar: string;
      location: string;
      rating: number;
      certified: boolean;
      mode: string;
      userPhone: string;
      orderCount: number;
    }

    interface PhotographerDetail extends Photographer {
      description: string;
      mutualIntro: string;
      userId: number | null;
      worksCount: number;
      reviewsCount: number;
    }

    interface AuditLog {
      id: number;
      adminId: number;
      adminName: string;
      method: string;
      path: string;
      status: number;
      createdAt: string;
    }

    interface AuditLogList {
      list: AuditLog[];
      total: number;
    }

    type OrderStatus = 'pending' | 'confirmed' | 'completed' | 'cancelled';

    interface Order {
      id: number;
      status: OrderStatus;
      date: string;
      time: string;
      price: number;
      createdAt: string;
      coserName: string;
      photographerName: string;
    }

    interface OrderList {
      list: Order[];
      total: number;
    }

    interface EventItem {
      id: number;
      name: string;
      location: string;
      venue: string;
      startDate: string;
      endDate: string;
      status: string;
      typeName: string;
      delFlag: boolean;
    }

    type UserStatus = 'active' | 'disabled';

    interface AdminUser {
      id: number;
      phone: string;
      name: string;
      avatar: string;
      role: 'photographer' | 'coser';
      status: UserStatus;
      createdAt: string;
      photographerId?: number | null;
    }

    interface UserList {
      list: AdminUser[];
      total: number;
    }

    interface AdminUserBooking {
      id: number;
      status: string;
      date: string;
      time: string;
      price: number;
      photographerName: string;
      serviceName: string;
    }

    interface AdminUserDetail extends AdminUser {
      stats: {
        bookingsCount: number;
        reviewsCount: number;
        favoritesCount: number;
        followsCount: number;
      };
      recentBookings: AdminUserBooking[];
    }

    type CertApplicationStatus = 'pending' | 'approved' | 'rejected';

    interface AdminCertApplication {
      id: number;
      userId: number;
      userName: string;
      photographerId: number;
      photographerName: string;
      evidenceImages: string[];
      evidenceDesc: string;
      status: CertApplicationStatus;
      createdAt: string;
      reviewReason?: string | null;
      adminId?: number | null;
      reviewedAt?: string | null;
    }

    interface CertApplicationList {
      list: AdminCertApplication[];
      total: number;
    }

    interface AdminBanner {
      id: number;
      imageUrl: string;
      title: string;
      linkType: string;
      linkId: number;
      sortOrder: number;
      isActive: boolean;
      createdAt: string;
    }

    type AdminRole = 'admin' | 'super';

    type AdminStatus = 'active' | 'disabled';

    interface AdminAccount {
      id: number;
      username: string;
      role: AdminRole;
      status: AdminStatus;
      createdAt: string;
    }

    type NotificationType = 'success' | 'info' | 'warning';

    type NotificationTargetType = 'all' | 'single';

    interface NotificationItem {
      id: number;
      type: NotificationType;
      title: string;
      content: string;
      userId: number | null;
      targetType: NotificationTargetType;
      createdAt: string;
    }

    interface NotificationList {
      list: NotificationItem[];
      total: number;
    }

    interface PublishNotificationRequest {
      type: NotificationType;
      title: string;
      content: string;
      targetType: NotificationTargetType;
      userId?: number;
    }

    type WorkStatus = 'active' | 'down';

    interface AdminWork {
      id: number;
      title: string;
      images: string[];
      photographerName: string;
      status: WorkStatus;
      createdAt: string;
    }

    interface AdminWorkList {
      list: AdminWork[];
      total: number;
    }

    interface AdminReview {
      id: number;
      userName: string;
      userAvatar: string;
      rating: number;
      content: string;
      photographerName: string;
      createdAt: string;
    }

    interface AdminReviewList {
      list: AdminReview[];
      total: number;
    }

    interface AdminTag {
      id: number;
      name: string;
      usageCount: number;
    }
  }
}
