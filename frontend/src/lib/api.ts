/* ── API client with automatic token refresh ───────────────────────────── */

import { Auth } from './auth';

interface ApiOptions extends RequestInit {
  _refreshed?: boolean;
}

export interface ApiResponse<T = unknown> {
  ok: boolean;
  status: number;
  body: T | null;
}

export async function api(
  path: string,
  options: ApiOptions = {},
): Promise<Response> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((options.headers as Record<string, string>) || {}),
  };
  if (Auth.access) headers['Authorization'] = `Bearer ${Auth.access}`;

  const init: ApiOptions = { ...options, headers };
  const refreshed = init._refreshed;
  delete init._refreshed;

  let res = await fetch(path, init);

  if (res.status === 401 && Auth.refresh && !refreshed) {
    const rr = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        refresh_token: Auth.refresh,
        device_name: 'web',
      }),
    });
    if (rr.ok) {
      const j = (await rr.json()) as {
        access_token: string;
        refresh_token: string;
      };
      Auth.set(j.access_token, j.refresh_token);
      return api(path, { ...options, _refreshed: true });
    }
    Auth.clear();
  }

  return res;
}

export async function apiJSON<T = unknown>(
  path: string,
  options: ApiOptions = {},
): Promise<ApiResponse<T>> {
  const res = await api(path, options);
  const text = await res.text();
  let body: T | null = null;
  if (text) {
    try {
      body = JSON.parse(text) as T;
    } catch {
      body = { raw: text } as unknown as T;
    }
  }
  return { ok: res.ok, status: res.status, body };
}
