import client from './client';
import type { CreateListenerPayload, Listener } from '../types';

export async function getListeners(): Promise<Listener[]> {
  const { data } = await client.get<Listener[]>('/listeners');
  return data;
}

export async function createListener(payload: CreateListenerPayload): Promise<Listener> {
  const { data } = await client.post<Listener>('/listeners', payload);
  return data;
}

export async function getListener(id: string): Promise<Listener> {
  const { data } = await client.get<Listener>(`/listeners/${id}`);
  return data;
}

export async function deleteListener(id: string): Promise<void> {
  await client.delete(`/listeners/${id}`);
}

export async function startListener(id: string): Promise<void> {
  await client.post(`/listeners/${id}/start`);
}

export async function stopListener(id: string): Promise<void> {
  await client.post(`/listeners/${id}/stop`);
}
