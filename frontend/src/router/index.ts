import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    { path: '/', redirect: '/app/overview' },
    { path: '/login', redirect: '/auth/login' },
    { path: '/auth', component: () => import('@/layouts/AuthLayout.vue'), children: [
      { path: 'login', component: () => import('@/pages/auth/LoginPage.vue') },
      { path: 'register', component: () => import('@/pages/auth/RegisterPage.vue') },
      { path: 'verify-email', component: () => import('@/pages/auth/VerifyEmailPage.vue') },
      { path: 'forgot-password', component: () => import('@/pages/auth/ForgotPasswordPage.vue') },
      { path: 'reset-password', component: () => import('@/pages/auth/ResetPasswordPage.vue') },
      { path: 'payment-result', component: () => import('@/pages/auth/PaymentResultPage.vue') },
      { path: 'terms', component: () => import('@/pages/auth/TermsPage.vue') },
      { path: 'privacy', component: () => import('@/pages/auth/PrivacyPage.vue') },
      { path: 'frozen', component: () => import('@/pages/auth/FrozenPage.vue') },
      { path: 'maintenance', component: () => import('@/pages/auth/MaintenancePage.vue') },
      { path: '404', component: () => import('@/pages/auth/NotFoundPage.vue') },
    ] },
    { path: '/app', component: () => import('@/layouts/ClientLayout.vue'), redirect: '/app/overview', children: [
      { path: 'overview', component: () => import('@/pages/client/OverviewPage.vue'), meta: { surface: 'client', title: '首页概览' } },
      { path: 'create', component: () => import('@/pages/client/CreatePage.vue'), meta: { surface: 'client', title: '创作' } },
      { path: 'garden', component: () => import('@/pages/client/GardenPage.vue'), meta: { surface: 'client', title: '创意园' } },
      { path: 'works', component: () => import('@/pages/client/WorksPage.vue'), meta: { surface: 'client', title: '作品' } },
      { path: 'assets', component: () => import('@/pages/client/AssetsPage.vue'), meta: { surface: 'client', title: '资产' } },
      { path: 'wallet', component: () => import('@/pages/client/WalletPage.vue'), meta: { surface: 'client', title: '钱包' } },
      { path: 'tickets', component: () => import('@/pages/client/TicketsPage.vue'), meta: { surface: 'client', title: '帮助与工单' } },
      { path: 'profile', component: () => import('@/pages/client/ProfilePage.vue'), meta: { surface: 'client', title: '个人资料' } },
    ] },
    { path: '/admin', component: () => import('@/layouts/AdminLayout.vue'), redirect: '/admin/overview', children: [
      { path: 'overview', component: () => import('@/pages/admin/OverviewPage.vue'), meta: { surface: 'admin', title: '首页概览' } },
      { path: 'workflows', component: () => import('@/pages/admin/WorkflowsPage.vue'), meta: { surface: 'admin', title: '工作流' } },
      { path: 'providers', component: () => import('@/pages/admin/ProvidersPage.vue'), meta: { surface: 'admin', title: '模型与供应商' } },
      { path: 'tasks', component: () => import('@/pages/admin/TasksPage.vue'), meta: { surface: 'admin', title: '任务中心' } },
      { path: 'monitoring', component: () => import('@/pages/admin/MonitoringPage.vue'), meta: { surface: 'admin', title: '运维监控' } },
      { path: 'pricing', component: () => import('@/pages/admin/PricingPage.vue'), meta: { surface: 'admin', title: '定价规则' } },
      { path: 'products', component: () => import('@/pages/admin/ProductsPage.vue'), meta: { surface: 'admin', title: '充值商品' } },
      { path: 'orders', component: () => import('@/pages/admin/OrdersPage.vue'), meta: { surface: 'admin', title: '订单管理' } },
      { path: 'users', component: () => import('@/pages/admin/UsersPage.vue'), meta: { surface: 'admin', title: '用户管理' } },
      { path: 'tickets', component: () => import('@/pages/admin/AdminTicketsPage.vue'), meta: { surface: 'admin', title: '工单中心' } },
      { path: 'announcements', component: () => import('@/pages/admin/AnnouncementsPage.vue'), meta: { surface: 'admin', title: '公告' } },
      { path: 'audit-log', component: () => import('@/pages/admin/AuditLogPage.vue'), meta: { surface: 'admin', title: '操作日志' } },
      { path: 'settings', component: () => import('@/pages/admin/SettingsPage.vue'), meta: { surface: 'admin', title: '系统设置' } },
    ] },
    { path: '/:pathMatch(.*)*', redirect: '/auth/404' },
  ],
})

export default router
