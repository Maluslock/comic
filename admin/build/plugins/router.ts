import type { RouteMeta } from 'vue-router';
import ElegantVueRouter from '@elegant-router/vue/vite';
import type { RouteKey } from '@elegant-router/types';

export function setupElegantRouter() {
  return ElegantVueRouter({
    layouts: {
      base: 'src/layouts/base-layout/index.vue',
      blank: 'src/layouts/blank-layout/index.vue'
    },
    routePathTransformer(routeName, routePath) {
      const key = routeName as RouteKey;

      if (key === 'login') {
        return '/login';
      }

      return routePath;
    },
    onRouteMetaGen(routeName) {
      const key = routeName as RouteKey;

      const constantRoutes: RouteKey[] = ['login', '403', '404', '500'];

      const menuIcons: Partial<Record<RouteKey, string>> = {
        dashboard: 'mdi:monitor-dashboard',
        photographer: 'mdi:account-check',
        order: 'mdi:clipboard-list',
        event: 'mdi:calendar-star',
        user: 'mdi:account-group',
        certification: 'mdi:star-check-outline',
        banner: 'mdi:image-multiple-outline',
        'admin-accounts': 'mdi:account-cog-outline',
        notification: 'mdi:bell-outline',
        content: 'mdi:book-open-page-variant-outline',
        'audit-log': 'mdi:history'
      };

      const meta: Partial<RouteMeta> = {
        title: key,
        i18nKey: `route.${key}` as App.I18n.I18nKey
      };

      if (menuIcons[key]) {
        meta.icon = menuIcons[key];
      }

      if (constantRoutes.includes(key)) {
        meta.constant = true;
      }

      return meta;
    }
  });
}
