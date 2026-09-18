declare namespace Api {
  /**
   * namespace Auth
   *
   * backend api module: "auth"
   */
  namespace Auth {
    /** Admin account returned by the login api */
    interface AdminUser {
      id: number;
      username: string;
      role: string;
    }

    interface LoginToken {
      token: string;
      admin: AdminUser;
    }

    interface UserInfo {
      userId: string;
      userName: string;
      roles: string[];
      buttons: string[];
    }
  }
}
