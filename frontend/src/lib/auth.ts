/* ── JWT parsing & Auth state (localStorage) ───────────────────────────── */

interface JWTClaims {
  sub?: string;
  role?: string;
  user_role?: string;
  exp?: number;
  iat?: number;
  [key: string]: unknown;
}

function parseJWT(token: string): JWTClaims | null {
  try {
    const payload = token.split('.')[1];
    const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(json) as JWTClaims;
  } catch {
    return null;
  }
}

export const Auth = {
  get access(): string | null {
    return localStorage.getItem('access_token');
  },

  get refresh(): string | null {
    return localStorage.getItem('refresh_token');
  },

  set(access: string, refresh: string): void {
    localStorage.setItem('access_token', access);
    localStorage.setItem('refresh_token', refresh);
  },

  clear(): void {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('username');
  },

  setUsername(u: string): void {
    localStorage.setItem('username', u);
  },

  get username(): string {
    return localStorage.getItem('username') || '';
  },

  claims(): JWTClaims | null {
    const t = this.access;
    return t ? parseJWT(t) : null;
  },

  role(): string | null {
    const c = this.claims();
    return (c && ((c.role as string) || (c.user_role as string))) || null;
  },

  isAdmin(): boolean {
    return this.role() === 'admin';
  },

  isAuthed(): boolean {
    return !!this.access;
  },
};
