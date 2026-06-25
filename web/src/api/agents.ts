import client from './client';
import type { Agent } from '../types';

export async function getAgents(status?: string): Promise<Agent[]> {
  const params = status ? { status } : {};
  const { data } = await client.get<Agent[]>('/agents', { params });
  return data;
}

export async function getAgent(id: string): Promise<Agent> {
  const { data } = await client.get<Agent>(`/agents/${id}`);
  return data;
}

export async function deleteAgent(id: string): Promise<void> {
  await client.delete(`/agents/${id}`);
}
