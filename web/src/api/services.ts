import { get, post, patch, del } from './client';
import type {
  Place,
  ResponsiblePerson,
  Facility,
  PlanTemplate,
  MaintenancePlan,
  PlanVersion,
  Execution,
  Anomaly,
  Todo,
  Notification,
  AuditLog,
  TimelineEvent,
  NextDateInfo,
  PageResult,
  FacilityStatus,
} from '../types/models';

export interface ListParams {
  limit?: number;
  offset?: number;
  order_by?: string;
  order?: 'asc' | 'desc';
  [key: string]: string | number | undefined;
}

function toQuery(params: ListParams): string {
  const usp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === '') continue;
    usp.set(k, String(v));
  }
  const s = usp.toString();
  return s ? `?${s}` : '';
}

// --- Places ---

export const PlaceService = {
  list: (params: ListParams = {}) => get<PageResult<Place>>(`/places${toQuery(params)}`),
  get: (id: string) => get<Place>(`/places/${id}`),
  create: (body: any) => post<Place>(`/places`, body),
  update: (id: string, body: any) => patch<Place>(`/places/${id}`, body),
  remove: (id: string) => del(`/places/${id}`),
};

// --- Responsible persons ---

export const PersonService = {
  list: (params: ListParams = {}) => get<PageResult<ResponsiblePerson>>(`/responsible-persons${toQuery(params)}`),
  get: (id: string) => get<ResponsiblePerson>(`/responsible-persons/${id}`),
  create: (body: any) => post<ResponsiblePerson>(`/responsible-persons`, body),
  update: (id: string, body: any) => patch<ResponsiblePerson>(`/responsible-persons/${id}`, body),
  setActive: (id: string, active: boolean) => post<ResponsiblePerson>(`/responsible-persons/${id}/active`, { active }),
  remove: (id: string) => del(`/responsible-persons/${id}`),
};

// --- Facilities ---

export const FacilityService = {
  list: (params: ListParams = {}) => get<PageResult<Facility>>(`/facilities${toQuery(params)}`),
  get: (id: string) => get<Facility>(`/facilities/${id}`),
  create: (body: any) => post<Facility>(`/facilities`, body),
  update: (id: string, body: any) => patch<Facility>(`/facilities/${id}`, body),
  transition: (id: string, to: FacilityStatus, reason: string) =>
    post<Facility>(`/facilities/${id}/transition`, { to, reason }),
  nextDate: (id: string) => get<NextDateInfo>(`/facilities/${id}/next-date`),
  timeline: (id: string, limit = 50) => get<TimelineEvent[]>(`/facilities/${id}/timeline?limit=${limit}`),
};

// --- Plan templates ---

export const TemplateService = {
  list: (params: ListParams = {}) => get<PageResult<PlanTemplate>>(`/plan-templates${toQuery(params)}`),
  get: (id: string) => get<PlanTemplate>(`/plan-templates/${id}`),
  create: (body: any) => post<PlanTemplate>(`/plan-templates`, body),
  update: (id: string, body: any) => patch<PlanTemplate>(`/plan-templates/${id}`, body),
};

// --- Plans ---

export const PlanService = {
  list: (params: ListParams = {}) => get<PageResult<MaintenancePlan>>(`/plans${toQuery(params)}`),
  get: (id: string) => get<MaintenancePlan>(`/plans/${id}`),
  create: (body: any) => post<MaintenancePlan>(`/plans`, body),
  changeCycle: (id: string, cycleDays: number, reason: string) =>
    patch<MaintenancePlan>(`/plans/${id}/cycle`, { cycle_days: cycleDays, reason }),
  skip: (id: string, reason: string) => post<MaintenancePlan>(`/plans/${id}/skip`, { reason }),
  regenerate: (id: string, reason: string) => post<MaintenancePlan>(`/plans/${id}/regenerate`, { reason }),
  assign: (id: string, personId: string, reason: string) =>
    post<MaintenancePlan>(`/plans/${id}/assign`, { person_id: personId, reason }),
  versions: (id: string) => get<PlanVersion[]>(`/plans/${id}/versions`),
};

// --- Executions ---

export const ExecutionService = {
  list: (params: ListParams = {}) => get<PageResult<Execution>>(`/executions${toQuery(params)}`),
  get: (id: string) => get<Execution>(`/executions/${id}`),
  submit: (body: any) => post<Execution>(`/executions`, body),
  review: (id: string, comment: string, approved: boolean) =>
    post<Execution>(`/executions/${id}/review`, { comment, approved }),
};

// --- Anomalies ---

export const AnomalyService = {
  list: (params: ListParams = {}) => get<PageResult<Anomaly>>(`/anomalies${toQuery(params)}`),
  get: (id: string) => get<Anomaly>(`/anomalies/${id}`),
  discover: (body: any) => post<Anomaly>(`/anomalies`, body),
  rectify: (id: string, measure: string) => post<Anomaly>(`/anomalies/${id}/rectify`, { measure }),
  reinspect: (id: string, result: string, pass: boolean, at?: string) =>
    post<Anomaly>(`/anomalies/${id}/reinspect`, { result, pass, at }),
  recover: (id: string) => post<Anomaly>(`/anomalies/${id}/recover`, {}),
};

// --- Todos ---

export const TodoService = {
  list: (params: ListParams = {}) => get<PageResult<Todo>>(`/todos${toQuery(params)}`),
  listByUser: (userId: string, params: ListParams = {}) =>
    get<PageResult<Todo>>(`/todos/by-user/${userId}${toQuery(params)}`),
  complete: (id: string) => patch<Todo>(`/todos/${id}/complete`, {}),
  skip: (id: string, reason: string) => patch<Todo>(`/todos/${id}/skip`, { reason }),
  reassign: (id: string, personId: string, reason: string) =>
    patch<Todo>(`/todos/${id}/reassign`, { person_id: personId, reason }),
};

// --- Notifications ---

export const NotificationService = {
  list: (params: ListParams = {}) => get<PageResult<Notification>>(`/notifications${toQuery(params)}`),
  listByUser: (userId: string, params: ListParams = {}) =>
    get<PageResult<Notification>>(`/notifications/by-user/${userId}${toQuery(params)}`),
  listUnread: (userId: string) => get<Notification[]>(`/notifications/unread/${userId}`),
  markRead: (id: string) => patch<void>(`/notifications/${id}/read`, {}),
};

// --- Audit ---

export const AuditService = {
  list: (params: ListParams = {}) => get<PageResult<AuditLog>>(`/audit-logs${toQuery(params)}`),
};

// --- Scheduler ---

export const SchedulerService = {
  scan: () => post<any>(`/scheduler/scan`, {}),
};
