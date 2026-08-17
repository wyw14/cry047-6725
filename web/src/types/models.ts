// Shared types matching the backend domain types.

export type PlaceType = 'library' | 'exhibit' | 'museum' | 'community' | 'other';
export type Criticality = 'critical' | 'important' | 'standard';
export type FacilityStatus =
  | 'normal'
  | 'pending_maintenance'
  | 'overdue'
  | 'restricted_use'
  | 'under_repair'
  | 'recovered';
export type AnomalySeverity = 'low' | 'medium' | 'high' | 'critical';
export type AnomalyStatus = 'open' | 'rectifying' | 'reinspecting' | 'recovered' | 'closed_no_action';
export type TodoType = 'maintenance' | 'rectification' | 'reinspection' | 'recovery' | 'review';
export type TodoStatus = 'open' | 'assigned' | 'completed' | 'skipped' | 'overdue';
export type ExecutionStatus = 'draft' | 'submitted' | 'reviewed' | 'rejected';
export type NotificationType =
  | 'maintenance_due'
  | 'maintenance_overdue'
  | 'anomaly_discovered'
  | 'anomaly_recovered'
  | 'plan_updated'
  | 'assigned';

export type Role = 'viewer' | 'operator' | 'engineer' | 'supervisor' | 'admin';

export interface Actor {
  id: string;
  name: string;
  role: Role;
}

export interface Place {
  id: string;
  name: string;
  type: PlaceType;
  address: string;
  code: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface ResponsiblePerson {
  id: string;
  name: string;
  email: string;
  phone: string;
  department: string;
  active: boolean;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface Facility {
  id: string;
  place_id: string;
  name: string;
  code: string;
  category: string;
  responsible_person_id: string;
  criticality: Criticality;
  status: FacilityStatus;
  description: string;
  alternative_facility_id?: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface InspectionItem {
  code: string;
  name: string;
  required: boolean;
  unit?: string;
  min_value?: number;
  max_value?: number;
}

export interface ConsumableSpec {
  code: string;
  name: string;
  quantity: number;
  unit?: string;
}

export interface PlanTemplate {
  id: string;
  name: string;
  cycle_days: number;
  inspection_items: InspectionItem[];
  consumables: ConsumableSpec[];
  requires_shutdown: boolean;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface MaintenancePlan {
  id: string;
  facility_id: string;
  template_id: string;
  current_cycle_days: number;
  next_due_date: string;
  last_executed_at?: string;
  last_execution_id?: string;
  active: boolean;
  skipped_count: number;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface PlanVersion {
  id: string;
  plan_id: string;
  version_number: number;
  cycle_days: number;
  template_id?: string;
  changed_by: string;
  changed_at: string;
  reason: string;
}

export interface InspectionValue {
  item_code: string;
  value?: string;
  numeric_value?: number;
  pass: boolean;
  note?: string;
}

export interface ConsumableConsumption {
  code: string;
  quantity: number;
}

export interface PhotoAttachment {
  path: string;
  mime_type: string;
  size: number;
  checksum?: string;
}

export interface Execution {
  id: string;
  plan_id: string;
  facility_id: string;
  executed_by: string;
  executed_at: string;
  inspection_values: InspectionValue[];
  photos: PhotoAttachment[];
  consumables_consumed: ConsumableConsumption[];
  review_comment?: string;
  reviewed_by?: string;
  reviewed_at?: string;
  status: ExecutionStatus;
  idempotency_key?: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface Anomaly {
  id: string;
  facility_id: string;
  execution_id?: string;
  discovered_by: string;
  discovered_at: string;
  description: string;
  severity: AnomalySeverity;
  status: AnomalyStatus;
  rectification_measure?: string;
  rectified_by?: string;
  rectified_at?: string;
  reinspection_result?: string;
  reinspected_by?: string;
  reinspected_at?: string;
  recovered_by?: string;
  recovered_at?: string;
  idempotency_key?: string;
  created_at: string;
  updated_at: string;
  version: number;
}

export interface Todo {
  id: string;
  facility_id: string;
  plan_id?: string;
  anomaly_id?: string;
  assigned_to: string;
  type: TodoType;
  due_date: string;
  status: TodoStatus;
  title: string;
  idempotency_key?: string;
  created_at: string;
  completed_at?: string;
  version: number;
}

export interface Notification {
  id: string;
  user_id: string;
  type: NotificationType;
  title: string;
  body: string;
  read_at?: string;
  created_at: string;
}

export interface AuditLog {
  id: string;
  action: string;
  entity_type: string;
  entity_id: string;
  actor_id: string;
  actor_name: string;
  before_state?: any;
  after_state?: any;
  request_id: string;
  reason?: string;
  created_at: string;
}

export interface TimelineEvent {
  id: string;
  facility_id: string;
  occurred_at: string;
  event_type: string;
  title: string;
  actor: string;
  detail?: any;
}

export interface NextDateInfo {
  facility_id: string;
  next_due_date: string;
  overdue_risk: boolean;
  days_until_due: number;
  last_executed_at?: string;
  alternative_facility_id?: string;
}

export interface PageResult<T> {
  items: T[];
  total: number;
  limit: number;
  page: number;
}

export interface ApiError {
  code: string;
  message: string;
  errors?: { field: string; message: string }[];
  request_id?: string;
}

export interface ApiResponse<T> {
  code: string;
  message: string;
  data?: T;
  errors?: { field: string; message: string }[];
  request_id?: string;
}
