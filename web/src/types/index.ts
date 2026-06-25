export interface User {
  id: string;
  username: string;
  role: string;
}

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface RefreshResponse {
  access_token: string;
  expires_in: number;
}

export interface Listener {
  id: string;
  name: string;
  protocol: string;
  host: string;
  port: number;
  status: 'stopped' | 'running' | 'error';
  config_json?: string;
  created_at: string;
  started_at?: string;
}

export interface CreateListenerPayload {
  name: string;
  protocol: string;
  host: string;
  port: number;
}

export interface Agent {
  id: string;
  listener_id?: string;
  hostname: string;
  os: string;
  arch: string;
  username: string;
  pid: number;
  local_ips?: string;
  version: string;
  first_seen: string;
  last_seen: string;
  heartbeat_interval: number;
  status: 'active' | 'inactive' | 'dead';
  metadata_json?: string;
}

export interface Task {
  id: string;
  agent_id: string;
  operator_id?: string;
  command: string;
  arguments: string;
  status: 'pending' | 'sent' | 'completed' | 'error' | 'cancelled';
  output?: string;
  error_message?: string;
  timeout_sec: number;
  created_at: string;
  sent_at?: string;
  completed_at?: string;
  duration_ms?: number;
}

export interface CreateTaskPayload {
  command: string;
  arguments?: string;
  timeout?: number;
}

export interface DashboardStats {
  agent_count: number;
  active_agents: number;
  listener_count: number;
  running_listeners: number;
  task_count: number;
  tasks_today: number;
}

export interface AuditLogEntry {
  id: string;
  operator_id?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  details_json?: string;
  ip_address?: string;
  created_at: string;
}

export interface BuildPayload {
  os: string;
  arch: string;
  listener_id: string;
  connection_type: string;
}

export interface BuildResponse {
  download_url: string;
  filename: string;
  checksum: string;
  os: string;
  arch: string;
  connection_type: string;
  listener_id: string;
}

export interface ApiError {
  error: {
    code: string;
    message: string;
  };
}

export interface PaginatedResponse<T> {
  data: T[];
  page: number;
  limit: number;
}
