import client from './client';
import type { BuildPayload, BuildResponse } from '../types';

export async function buildImplant(payload: BuildPayload): Promise<BuildResponse> {
  const { data } = await client.post<BuildResponse>('/build', payload);
  return data;
}
