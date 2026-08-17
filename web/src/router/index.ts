import { createRouter, createWebHistory } from 'vue-router';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/facilities' },
    { path: '/facilities', name: 'facilities', component: () => import('../views/FacilityLedger.vue'), meta: { title: '设施台账' } },
    { path: '/facilities/:id', name: 'facility-detail', component: () => import('../views/FacilityDetail.vue'), meta: { title: '设施详情' } },
    { path: '/calendar', name: 'calendar', component: () => import('../views/PlanCalendar.vue'), meta: { title: '计划日历' } },
    { path: '/workbench', name: 'workbench', component: () => import('../views/ExecutionWorkbench.vue'), meta: { title: '执行工作台' } },
    { path: '/anomalies', name: 'anomalies', component: () => import('../views/AnomalyRectification.vue'), meta: { title: '异常整改' } },
    { path: '/todos', name: 'todos', component: () => import('../views/PersonalTodos.vue'), meta: { title: '个人待办' } },
    { path: '/config', name: 'config', component: () => import('../views/ConfigPage.vue'), meta: { title: '配置页' } },
    { path: '/notifications', name: 'notifications', component: () => import('../views/Notifications.vue'), meta: { title: '提醒中心' } },
    { path: '/audit', name: 'audit', component: () => import('../views/AuditLogs.vue'), meta: { title: '审计日志' } },
  ],
});

router.afterEach((to) => {
  if (to.meta?.title) {
    document.title = `${to.meta.title} - 公共设施预防性保养协同平台`;
  }
});

export default router;
