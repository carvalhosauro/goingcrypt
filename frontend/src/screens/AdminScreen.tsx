import { useState, useEffect, type FC } from 'react';
import {
  Activity,
  Users,
  Settings,
  Lock,
  Eye,
  Clock,
} from 'lucide-react';
import Card from '../components/Card';
import Badge from '../components/Badge';
import Btn from '../components/Button';
import { api, apiJSON } from '../lib/api';
import styles from './AdminScreen.module.css';

interface AccessLog {
  id: string;
  slug: string;
  opened_at: string;
  ip_address: string;
  user_agent: string;
}

interface LinkEntry {
  slug: string;
  status: string;
  created_at: string;
}

interface UserEntry {
  id: string;
  username: string;
  role: string;
  banned: boolean;
  created_at: string;
}

interface AdminScreenProps {
  toast: (msg: string, kind?: 'info' | 'error' | 'success') => void;
}

interface FeedEvent {
  type: 'SECRET_ACCESSED' | 'SECRET_CREATED' | 'SECRET_EXPIRED';
  time: string;
  ip: string;
  ua: string;
  slug: string;
  id: string;
}

const actionCfg: Record<string, { color: 'green' | 'blue' | 'gray'; label: string }> = {
  SECRET_ACCESSED: { color: 'green', label: 'Acessado' },
  SECRET_CREATED:  { color: 'blue',  label: 'Criado' },
  SECRET_EXPIRED:  { color: 'gray',  label: 'Expirado' },
};

function fmtTime(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  return d.toLocaleTimeString('pt-BR', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

function fmtDate(iso?: string): string {
  return iso ? new Date(iso).toLocaleDateString('pt-BR') : '—';
}

const tabItems = [
  { id: 'audit',    label: 'Audit Log',     icon: Activity },
  { id: 'users',    label: 'Usuários',      icon: Users },
  { id: 'settings', label: 'Configurações', icon: Settings },
] as const;

type TabId = (typeof tabItems)[number]['id'];

const AdminScreen: FC<AdminScreenProps> = ({ toast }) => {
  const [tab, setTab] = useState<TabId>('audit');
  const [logs, setLogs] = useState<AccessLog[] | null>(null);
  const [users, setUsers] = useState<UserEntry[] | null>(null);
  const [links, setLinks] = useState<LinkEntry[] | null>(null);

  useEffect(() => {
    if (tab === 'audit' && logs === null) {
      apiJSON<{ logs: AccessLog[] }>('/api/v1/admin/access-logs').then(
        (res) => {
          if (res.ok) setLogs(res.body?.logs || []);
          else toast(`Falha audit logs: ${res.status}`, 'error');
        },
      );
      apiJSON<{ links: LinkEntry[] }>('/api/v1/admin/links').then((res) => {
        if (res.ok) setLinks(res.body?.links || []);
      });
    }
    if (tab === 'users' && users === null) {
      apiJSON<{ users: UserEntry[] }>('/api/v1/admin/users').then((res) => {
        if (res.ok) setUsers(res.body?.users || []);
        else toast(`Falha users: ${res.status}`, 'error');
      });
    }
  }, [tab, logs, users, toast]);

  const grantAdmin = async (userID: string) => {
    if (!confirm('Promover este usuário a admin?')) return;
    const res = await api(`/api/v1/admin/users/${userID}/grant-admin`, {
      method: 'POST',
    });
    if (res.status === 204) {
      toast('Usuário promovido', 'success');
      setUsers(null);
    } else {
      toast('Falha ao promover', 'error');
    }
  };

  // Build audit feed
  const feed: FeedEvent[] = (() => {
    const events: FeedEvent[] = [];
    (logs || []).forEach((l) =>
      events.push({
        type: 'SECRET_ACCESSED',
        time: l.opened_at,
        ip: l.ip_address,
        ua: l.user_agent,
        slug: l.slug,
        id: 'log-' + l.id,
      }),
    );
    (links || []).forEach((l) =>
      events.push({
        type: l.status === 'EXPIRED' ? 'SECRET_EXPIRED' : 'SECRET_CREATED',
        time: l.created_at,
        ip: '—',
        ua: '—',
        slug: l.slug,
        id: 'link-' + l.slug,
      }),
    );
    events.sort(
      (a, b) => new Date(b.time).getTime() - new Date(a.time).getTime(),
    );
    return events.slice(0, 80);
  })();

  const statItems = [
    {
      label: 'Total de links',
      value: links ? links.length : '—',
      color: '#4f6ef7',
      icon: Lock,
    },
    {
      label: 'Acessos',
      value: logs ? logs.length : '—',
      color: '#10b981',
      icon: Eye,
    },
    {
      label: 'Aguardando',
      value: links
        ? links.filter((l) => l.status === 'WAITING').length
        : '—',
      color: '#f59e0b',
      icon: Clock,
    },
    {
      label: 'Usuários',
      value: users ? users.length : '—',
      color: '#8b5cf6',
      icon: Users,
    },
  ];

  const settingsData = [
    {
      label: 'Tamanho máximo do secret',
      sub: 'Limite de caracteres por secret',
      val: '4096',
      unit: 'chars',
    },
    {
      label: 'Expiração padrão',
      sub: 'Quando nenhuma expiração é definida',
      val: '24',
      unit: 'horas',
    },
    {
      label: 'Secrets por usuário',
      sub: 'Limite total de secrets simultâneos',
      val: '50',
      unit: 'secrets',
    },
    {
      label: 'Janitor interval',
      sub: 'Frequência da limpeza de expirados',
      val: '60',
      unit: 'min',
    },
  ];

  return (
    <div className={styles.page}>
      <div className={styles.headerRow}>
        <div>
          <h1 className={styles.title}>Painel Admin</h1>
          <p className={styles.subtitle}>
            Auditoria completa e gestão da plataforma.
          </p>
        </div>
        <Badge color="red">ADMIN</Badge>
      </div>

      <div className={styles.statsGrid}>
        {statItems.map((s, i) => (
          <Card key={i}>
            <div className={styles.statCard}>
              <div
                className={styles.statIcon}
                style={{ background: `${s.color}18` }}
              >
                <s.icon size={15} color={s.color} />
              </div>
              <div>
                <div className={styles.statValue}>{s.value}</div>
                <div className={styles.statLabel}>{s.label}</div>
              </div>
            </div>
          </Card>
        ))}
      </div>

      <div className={styles.tabs}>
        {tabItems.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`${styles.tab} ${tab === t.id ? styles.active : ''}`}
          >
            <t.icon size={13} />
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'audit' && (
        <Card>
          <div className={styles.tableHeader}>
            <span className={styles.tableTitle}>Log de Auditoria</span>
            <span className={styles.tableSub}>Mais recentes</span>
          </div>
          <div className={styles.gridHeader}>
            {['Hora', 'IP + User-Agent', 'Ação', 'Slug'].map((h) => (
              <span key={h} className={styles.gridHeaderCell}>
                {h}
              </span>
            ))}
          </div>
          {logs === null && (
            <div className={styles.empty}>Carregando...</div>
          )}
          {feed.length === 0 && logs !== null && (
            <div className={styles.empty}>Sem eventos.</div>
          )}
          {feed.map((ev, i) => {
            const ac = actionCfg[ev.type] || {
              color: 'gray' as const,
              label: ev.type,
            };
            return (
              <div
                key={ev.id}
                className={`${styles.auditRow} ${i < feed.length - 1 ? styles.auditRowBorder : ''}`}
              >
                <div>
                  <div className={styles.auditTime}>{fmtTime(ev.time)}</div>
                  <div className={styles.auditDate}>{fmtDate(ev.time)}</div>
                </div>
                <div>
                  <div className={styles.auditIp}>{ev.ip}</div>
                  <div className={styles.auditUa}>{ev.ua}</div>
                </div>
                <Badge color={ac.color}>{ac.label}</Badge>
                <span className={styles.auditSlug}>{ev.slug}</span>
              </div>
            );
          })}
        </Card>
      )}

      {tab === 'users' && (
        <Card>
          <div className={styles.tableHeader}>
            <span className={styles.tableTitle}>Gerenciar Usuários</span>
          </div>
          {users === null && (
            <div className={styles.empty}>Carregando...</div>
          )}
          {users &&
            users.map((u, i) => (
              <div
                key={u.id}
                className={`${styles.userRow} ${i < users.length - 1 ? styles.userRowBorder : ''}`}
              >
                <div
                  className={styles.userAvatar}
                  style={{
                    background: `hsl(${i * 70 + 200},40%,35%)`,
                    color: `hsl(${i * 70 + 200},60%,75%)`,
                  }}
                >
                  {u.username[0].toUpperCase()}
                </div>
                <div className={styles.userBody}>
                  <div className={styles.userName}>{u.username}</div>
                  <div className={styles.userId}>{u.id.slice(0, 8)}…</div>
                </div>
                <Badge color={u.role === 'admin' ? 'red' : 'blue'}>
                  {u.role}
                </Badge>
                <span className={styles.userDate}>
                  {fmtDate(u.created_at)}
                </span>
                <Badge color={u.banned ? 'red' : 'green'} dot>
                  {u.banned ? 'banido' : 'ativo'}
                </Badge>
                <div className={styles.userActions}>
                  {u.role !== 'admin' && (
                    <Btn
                      variant="secondary"
                      size="sm"
                      onClick={() => grantAdmin(u.id)}
                    >
                      Promover
                    </Btn>
                  )}
                </div>
              </div>
            ))}
        </Card>
      )}

      {tab === 'settings' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {settingsData.map((s, i) => (
            <Card key={i}>
              <div className={styles.settingsRow}>
                <div>
                  <div className={styles.settingLabel}>{s.label}</div>
                  <div className={styles.settingSub}>{s.sub}</div>
                </div>
                <div className={styles.settingInputWrap}>
                  <input
                    defaultValue={s.val}
                    className={styles.settingInput}
                  />
                  <span className={styles.settingUnit}>{s.unit}</span>
                </div>
              </div>
            </Card>
          ))}
          <div className={styles.settingsNote}>
            Configurações de runtime (placeholder — endpoint não implementado).
          </div>
        </div>
      )}
    </div>
  );
};

export default AdminScreen;
