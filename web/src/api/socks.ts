import client from './client';
import type { SocksStatus } from '../types';

export async function startSocks(agentId: string, port: number): Promise<{ message: string; port: number }> {
  const { data } = await client.post(`/agents/${agentId}/socks`, { port });
  return data;
}

export async function getSocksStatus(agentId: string): Promise<SocksStatus> {
  const { data } = await client.get<SocksStatus>(`/agents/${agentId}/socks`);
  return data;
}

export async function stopSocks(agentId: string): Promise<void> {
  await client.delete(`/agents/${agentId}/socks`);
}
