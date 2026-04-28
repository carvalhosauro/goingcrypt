import { useState, useEffect, useCallback, type FC } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Lock,
  CheckCircle,
  Clock,
  X,
  List,
  Plus,
  Trash2,
} from 'lucide-react';
import Card from '../components/Card';
import Badge from '../components/Badge';
import Btn from '../components/Button';
import Spinner from '../components/Spinner';
import { api, apiJSON } from '../lib/api';
import styles from './SecretsScreen.module.css';

interface Secret {
  slug: string;
  status: 'WAITING' | 'OPENED' | 'EXPIRED';
  created_at: string;
  expires_at?: string;
}

interface SecretsScreenProps {
  toast: (msg: string, kind?: 'info' | 'error' | 'success') => void;
}

const stCfg: Record<
  string,
  { color: 'blue' | 'green' | 'gray'; icon: typeof Lock; label: string }
> = {
  WAITING: { color: 'blue', icon: Lock, label: 'Aguardando' },
  OPENED: { color: 'green', icon: CheckCircle, label: 'Acessado' },
  EXPIRED: { color: 'gray', icon: Clock, label: 'Expirado' },
};

function fmtDate(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  const now = new Date();
  const diff = (now.getTime() - d.getTime()) / 1000;
  if (diff < 60) return 'agora';
  if (diff < 3600) return `há ${Math.floor(diff / 60)}min`;
  if (diff < 86400) return `há ${Math.floor(diff / 3600)}h`;
  return `há ${Math.floor(diff / 86400)}d`;
}

const SecretsScreen: FC<SecretsScreenProps> = ({ toast }) => {
  const navigate = useNavigate();
  const [secrets, setSecrets] = useState<Secret[] | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setBusy(true);
    const res = await apiJSON<{ links: Secret[]; error?: string }>(
      '/api/v1/links/',
    );
    setBusy(false);
    if (!res.ok) {
      toast(
        `Falha ao carregar: ${res.body?.error || res.status}`,
        'error',
      );
      return;
    }
    setSecrets(res.body?.links || []);
  }, [toast]);

  useEffect(() => {
    load();
  }, [load]);

  const counts = (secrets || []).reduce(
    (acc, s) => {
      acc[s.status] = (acc[s.status] || 0) + 1;
      return acc;
    },
    {} as Record<string, number>,
  );

  const invalidate = async (slug: string) => {
    if (!confirm(`Invalidar segredo ${slug}?`)) return;
    const res = await api(`/api/v1/links/${encodeURIComponent(slug)}`, {
      method: 'DELETE',
    });
    if (res.status === 204) {
      toast('Segredo invalidado', 'success');
      load();
    } else {
      toast('Falha ao invalidar', 'error');
    }
  };

  const stats = [
    {
      label: 'Total criados',
      value: secrets ? secrets.length : '—',
      color: '#4f6ef7',
      icon: List,
    },
    {
      label: 'Aguardando',
      value: counts.WAITING || 0,
      color: '#818cf8',
      icon: Clock,
    },
    {
      label: 'Acessados',
      value: counts.OPENED || 0,
      color: '#10b981',
      icon: CheckCircle,
    },
    {
      label: 'Expirados',
      value: counts.EXPIRED || 0,
      color: '#64748b',
      icon: X,
    },
  ];

  return (
    <div className={styles.page}>
      <div className={styles.headerRow}>
        <div>
          <h1 className={styles.title}>Meus Segredos</h1>
          <p className={styles.subtitle}>
            Histórico e status dos segredos que você criou.
          </p>
        </div>
        <Btn onClick={() => navigate('/')} variant="primary" icon={Plus} size="sm">
          Novo Segredo
        </Btn>
      </div>

      <div className={styles.statsGrid}>
        {stats.map((s, i) => (
          <Card key={i} style={{ padding: '15px 17px' }}>
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

      <Card>
        <div className={styles.tableHeader}>
          <Lock size={13} color="#4f6ef7" />
          <span className={styles.tableTitle}>Secrets</span>
          {busy && (
            <span style={{ marginLeft: 'auto' }}>
              <Spinner size={14} />
            </span>
          )}
        </div>

        {secrets === null && (
          <div className={styles.empty}>Carregando...</div>
        )}
        {secrets !== null && secrets.length === 0 && (
          <div className={styles.empty}>
            Nenhum segredo ainda.{' '}
            <button className={styles.emptyLink} onClick={() => navigate('/')}>
              Crie o primeiro
            </button>
            .
          </div>
        )}
        {secrets &&
          secrets.map((s, i) => {
            const st = stCfg[s.status] || {
              color: 'gray' as const,
              icon: Lock,
              label: s.status,
            };
            const StIcon = st.icon;
            const iconBg =
              s.status === 'OPENED'
                ? 'var(--green-dim)'
                : s.status === 'EXPIRED'
                  ? 'rgba(100,116,139,0.1)'
                  : 'var(--blue-dim)';
            const iconColor =
              s.status === 'OPENED'
                ? '#10b981'
                : s.status === 'EXPIRED'
                  ? '#64748b'
                  : '#4f6ef7';

            return (
              <div
                key={s.slug}
                className={`${styles.row} ${i < secrets.length - 1 ? styles.rowBorder : ''}`}
              >
                <div className={styles.rowIcon} style={{ background: iconBg }}>
                  <StIcon size={15} color={iconColor} />
                </div>
                <div className={styles.rowBody}>
                  <div className={styles.rowSlug}>{s.slug}</div>
                  <div className={styles.rowMeta}>
                    <span className={styles.rowUrl}>
                      {window.location.host}/s/{s.slug}
                    </span>
                    <span className={styles.rowDot}>·</span>
                    <span className={styles.rowCreated}>
                      criado {fmtDate(s.created_at)}
                    </span>
                    {s.expires_at && (
                      <>
                        <span className={styles.rowDot}>·</span>
                        <span className={styles.rowExpires}>
                          expira{' '}
                          {new Date(s.expires_at).toLocaleString('pt-BR')}
                        </span>
                      </>
                    )}
                  </div>
                </div>
                <Badge color={st.color} dot>
                  {st.label}
                </Badge>
                {s.status === 'WAITING' && (
                  <Btn
                    onClick={() => invalidate(s.slug)}
                    variant="danger"
                    size="sm"
                    icon={Trash2}
                  >
                    Invalidar
                  </Btn>
                )}
              </div>
            );
          })}
      </Card>
    </div>
  );
};

export default SecretsScreen;
