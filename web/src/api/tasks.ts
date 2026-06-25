import client from './client';
import type { CreateTaskPayload, Task } from '../types';

export async function getTasks(agentId: string, status?: string): Promise<Task[]> {
  const params = status ? { status } : {};
  const { data } = await client.get<Task[]>(`/agents/${agentId}/tasks`, { params });
  return data;
}

export async function createTask(agentId: string, payload: CreateTaskPayload): Promise<Task> {
  const { data } = await client.post<Task>(`/agents/${agentId}/tasks`, payload);
  return data;
}

export async function getTask(taskId: string): Promise<Task> {
  const { data } = await client.get<Task>(`/tasks/${taskId}`);
  return data;
}

export async function deleteTask(taskId: string): Promise<void> {
  await client.delete(`/tasks/${taskId}`);
}
