import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { authReturnPath } from '@/shared/lib/auth-redirect'
import { getAuthSession, onAuthSessionChange } from '@/shared/lib/auth-session'

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: (to) => to.hash ? { el: to.hash, top: 24 } : { top: 0 },
  routes: [
    { path: '/', component: () => import('@/layouts/PublicLayout.vue'), meta: { surface: 'client' }, children: [
      { path: '', component: () => import('@/pages/public/ProductHomePage.vue'), meta: { title: '用镜头讲故事' } },
      { path: 'guides/:id', component: () => import('@/pages/public/GuideDetailPage.vue'), meta: { title: '镜头教程' } },
    ] },
    { path: '/login', redirect: '/auth/login' },
    { path: '/auth', component: () => import('@/layouts/AuthLayout.vue'), children: [
      { path: 'login', component: () => import('@/pages/auth/LoginPage.vue'), meta: { guest: true } },
      { path: 'register', component: () => import('@/pages/auth/RegisterPage.vue'), meta: { guest: true } },
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
    { path: '/app', component: () => import('@/layouts/ClientLayout.vue'), redirect: '/app/overview', meta: { requiresAuth: true }, children: [
      { path: 'overview', component: () => import('@/pages/client/OverviewPage.vue'), meta: { surface: 'client', title: '首页概览' } },
      { path: 'create', component: () => import('@/pages/client/CreatePage.vue'), meta: { surface: 'client', title: '创作' } },
      { path: 'garden', component: () => import('@/pages/client/GardenPage.vue'), meta: { surface: 'client', title: '广场' } },
      { path: 'works', component: () => import('@/pages/client/WorksPage.vue'), meta: { surface: 'client', title: '任务' } },
      { path: 'works/:id', component: () => import('@/pages/client/TaskDetailPage.vue'), meta: { surface: 'client', title: '任务详情' } },
      { path: 'assets', component: () => import('@/pages/client/AssetsPage.vue'), meta: { surface: 'client', title: '资产' } },
      { path: 'trash', component: () => import('@/pages/client/TrashPage.vue'), meta: { surface: 'client', title: '回收站' } },
      { path: 'wallet', component: () => import('@/pages/client/WalletPage.vue'), meta: { surface: 'client', title: '钱包' } },
      { path: 'tickets', component: () => import('@/pages/client/TicketsPage.vue'), meta: { surface: 'client', title: '帮助与工单' } },
      { path: 'profile', component: () => import('@/pages/client/ProfilePage.vue'), meta: { surface: 'client', title: '个人资料' } },
    ] },
    { path: '/admin', component: () => import('@/layouts/AdminLayout.vue'), redirect: '/admin/overview', meta: { requiresAuth: true, requiresAdmin: true }, children: [
      { path: 'overview', component: () => import('@/pages/admin/OverviewPage.vue'), meta: { surface: 'admin', title: '首页概览' } },
      { path: 'workflows', component: () => import('@/pages/admin/WorkflowsPage.vue'), meta: { surface: 'admin', title: '工作流' } },
      { path: 'carriers-models', component: () => import('@/pages/admin/CarrierModelsPage.vue'), meta: { surface: 'admin', title: '承运商' } },
      { path: 'supplier-models', component: () => import('@/pages/admin/SupplierModelsPage.vue'), meta: { surface: 'admin', title: '供应商' } },
      { path: 'providers', redirect: '/admin/supplier-models' },
      { path: 'tasks', component: () => import('@/pages/admin/TasksPage.vue'), meta: { surface: 'admin', title: '任务中心' } },
      { path: 'monitoring', component: () => import('@/pages/admin/MonitoringPage.vue'), meta: { surface: 'admin', title: '运维监控' } },
      { path: 'pricing', component: () => import('@/pages/admin/PricingPage.vue'), meta: { surface: 'admin', title: '价格审计' } },
      { path: 'tickets', component: () => import('@/pages/admin/AdminTicketsPage.vue'), meta: { surface: 'admin', title: '工单中心' } },
      { path: 'audit-log', component: () => import('@/pages/admin/AuditLogPage.vue'), meta: { surface: 'admin', title: '操作日志' } },
      { path: 'settings', component: () => import('@/pages/admin/SettingsPage.vue'), meta: { surface: 'admin', title: '系统设置' } },
    ] },
    { path: '/:pathMatch(.*)*', redirect: '/auth/404' },
  ],
})

router.beforeEach(async (to) => {
  const needsAuth = to.matched.some((r) => r.meta.requiresAuth)
  const needsAdmin = to.matched.some((r) => r.meta.requiresAdmin)
  const isGuest = to.matched.some((r) => r.meta.guest)
  const auth = useAuthStore()

  if (!auth.bootstrapped) {
    await auth.bootstrap()
  }

  const authenticated = auth.isAuthenticated
  if (needsAuth && !authenticated) {
    const reason = auth.bootstrapError ? 'session-unavailable' : getAuthSession().reason === 'expired' ? 'session-expired' : undefined
    return { path: '/auth/login', query: { redirect: authReturnPath(to.fullPath), reason } }
  }
  if (needsAdmin && auth.user?.role !== 'admin') {
    return '/app/overview'
  }
  if (isGuest && authenticated) {
    return authReturnPath(to.query.redirect)
  }
  return true
})

onAuthSessionChange((next, previous, source) => {
  const current = router.currentRoute.value
  // main.ts finishes bootstrap before installing the router. Until a protected
  // route has actually mounted, its guard owns the initial redirect.
  if (!current.matched.some((record) => record.meta.requiresAuth)) return
  if (source === 'storage' && next.id !== previous.id && next.accessToken) {
    // Another tab changed accounts. Reload private stores and re-check roles.
    router.go(0)
    return
  }
  if (!next.accessToken && (next.reason === 'expired' || next.reason === 'logout' || source === 'storage')) {
    void router.replace({
      path: '/auth/login',
      query: source === 'local' && next.reason === 'logout' ? {} : {
        redirect: authReturnPath(current.fullPath), reason: next.reason === 'expired' ? 'session-expired' : 'session-ended',
      },
    })
  }
})

export default router
