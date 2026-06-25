import client from './client';
import type { DashboardStats, AuditLogEntry, PaginatedResponse } from '../types';

export async function getStats(): Promise<DashboardStats> {
  const { data } = await client.get<DashboardStats>('/dashboard/stats');
  return data;
}

export async function getAuditLogs(page = 1, limit = 50): Promise<PaginatedResponse<AuditLogEntry>> {
  const { data } = await client.get<PaginatedResponse<AuditLogEntry>>('/audit', {
    params: { page, limit },
  });
  return data;
}
