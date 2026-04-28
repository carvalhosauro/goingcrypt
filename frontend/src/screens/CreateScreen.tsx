import { useState, type FC } from 'react';
import {
  ShieldCheck,
  Clock,
  Lock,
  Link as LinkIcon,
  Mail,
  Smartphone,
  Copy,
  Check,
  Flame,
  CheckCircle,
  Plus,
} from 'lucide-react';
import Card from '../components/Card';
import Btn from '../components/Button';
import { generateAesKey, exportKeyB64Url, encryptText } from '../lib/crypto';
import { apiJSON } from '../lib/api';
import styles from './CreateScreen.module.css';

interface CreateScreenProps {
  toast: (msg: string, kind?: 'info' | 'error' | 'success') => void;
}

const expiryOpts = [
  { v: '1h', l: '1 hora' },
  { v: '24h', l: '24 horas' },
  { v: '168h', l: '7 dias' },
  { v: '720h', l: '30 dias' },
];

const CreateScreen: FC<CreateScreenProps> = ({ toast }) => {
  const [content, setContent] = useState('');
  const [expiry, setExpiry] = useState('24h');
  const [step, setStep] = useState<'form' | 'encrypting' | 'done'>('form');
  const [copied1, setCopied1] = useState(false);
  const [copied2, setCopied2] = useState(false);
  const [progress, setProgress] = useState(0);
  const [result, setResult] = useState<{
    slug: string;
    keyB64Url: string;
  } | null>(null);

  const handleEncrypt = async () => {
    if (!content.trim()) return;
    setStep('encrypting');
    setProgress(10);
    try {
      const key = await generateAesKey();
      setProgress(35);
      const keyB64Url = await exportKeyB64Url(key);
      setProgress(55);
      const ciphered = await encryptText(content, key);
      setProgress(75);
      const res = await apiJSON<{ slug: string; error?: string }>(
        '/api/v1/links/',
        {
          method: 'POST',
          body: JSON.stringify({
            key: keyB64Url,
            ciphered_text: ciphered,
            expires_in: expiry,
          }),
        },
      );
      setProgress(100);
      if (!res.ok) throw new Error(res.body?.error || 'falha ao criar');
      await new Promise((r) => setTimeout(r, 250));
      setResult({ slug: res.body!.slug, keyB64Url });
      setStep('done');
    } catch (err) {
      toast(`Erro: ${err instanceof Error ? err.message : String(err)}`, 'error');
      setStep('form');
    }
  };

  const url = result ? `${window.location.origin}/s/${result.slug}` : '';
  const fragment = result ? `#${result.keyB64Url}` : '';
  const fullURL = url + fragment;

  const copy = async (text: string, onSuccess: () => void) => {
    try {
      await navigator.clipboard.writeText(text);
      onSuccess();
    } catch {
      toast('Falha ao copiar', 'error');
    }
  };

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <h1 className={styles.title}>Compartilhar Segredo</h1>
        <p className={styles.subtitle}>
          Cole texto sensível abaixo. Criptografado no navegador com AES-256-GCM
          — o servidor armazena apenas o hash da chave.
        </p>
      </div>

      {step === 'form' && (
        <div className={styles.animated}>
          <div className={styles.infoBanner}>
            <ShieldCheck size={16} color="#4f6ef7" strokeWidth={2} />
            <span className={styles.infoText}>
              A URL e a chave de decriptação são distribuídas em{' '}
              <strong>dois canais separados</strong>. O servidor armazena apenas{' '}
              <code>SHA-256(chave)</code> + cifra.
            </span>
          </div>

          <Card style={{ padding: 22, marginBottom: 14 }}>
            <label className={styles.fieldLabel}>Conteúdo Secreto</label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder="Cole aqui: senha, chave API, token, mensagem confidencial..."
              rows={7}
              className={styles.textarea}
            />
            <div className={styles.charCount}>
              <span className={content.length > 4096 ? styles.charCountOver : ''}>
                {content.length} / 4096
              </span>
            </div>
          </Card>

          <div style={{ marginBottom: 20 }}>
            <Card style={{ padding: 16 }}>
              <div className={styles.expiryLabel}>
                <Clock size={11} /> Expiração
              </div>
              <div className={styles.expiryOptions}>
                {expiryOpts.map((o) => (
                  <button
                    key={o.v}
                    onClick={() => setExpiry(o.v)}
                    className={`${styles.expiryBtn} ${expiry === o.v ? styles.active : ''}`}
                  >
                    {o.l}
                  </button>
                ))}
              </div>
              <div className={styles.expiryNote}>
                Por padrão segredos são de uso único — invalidados ao primeiro
                acesso.
              </div>
            </Card>
          </div>

          <Btn
            onClick={handleEncrypt}
            variant="primary"
            size="lg"
            icon={Lock}
            disabled={!content.trim() || content.length > 4096}
            style={{ width: '100%' }}
          >
            Criptografar e Gerar Link
          </Btn>
        </div>
      )}

      {step === 'encrypting' && (
        <div className={styles.encryptingWrap}>
          <div className={styles.spinnerLg} />
          <div style={{ textAlign: 'center' }}>
            <div className={styles.encryptTitle}>
              Criptografando no seu navegador...
            </div>
            <div className={styles.encryptSub}>
              AES-256-GCM · Web Crypto API · chave gerada localmente
            </div>
            <div className={styles.progressBar}>
              <div
                className={styles.progressFill}
                style={{ width: `${progress}%` }}
              />
            </div>
            <div className={styles.progressText}>
              {progress < 30
                ? 'Gerando chave 256-bit...'
                : progress < 60
                  ? 'Aplicando AES-256-GCM...'
                  : progress < 90
                    ? 'Calculando SHA-256(key)...'
                    : 'Enviando cifra ao servidor...'}
            </div>
          </div>
        </div>
      )}

      {step === 'done' && result && (
        <div className={styles.animated}>
          <div style={{ textAlign: 'center', marginBottom: 26 }}>
            <div className={styles.doneIcon}>
              <CheckCircle size={24} color="#10b981" strokeWidth={2} />
            </div>
            <div className={styles.doneTitle}>Segredo criado!</div>
            <div className={styles.doneSub}>
              Distribua os dois canais por meios{' '}
              <strong>diferentes</strong> (ex: e-mail + SMS) — ou use o link
              completo em um canal seguro.
            </div>
          </div>

          <Card
            style={{
              padding: 18,
              marginBottom: 14,
              borderColor: 'rgba(16,185,129,0.25)',
            }}
          >
            <div className={styles.fullLink}>
              <LinkIcon size={14} color="#10b981" />
              <span className={styles.linkLabel}>
                Link completo (URL + chave)
              </span>
            </div>
            <div className={styles.linkBox}>{fullURL}</div>
            <Btn
              onClick={() =>
                copy(fullURL, () => toast('Link completo copiado', 'success'))
              }
              variant="secondary"
              size="sm"
              icon={Copy}
              style={{ width: '100%' }}
            >
              Copiar Link Completo
            </Btn>
          </Card>

          <div className={styles.divider}>ou divida em dois canais</div>

          <div className={styles.channels}>
            <Card
              style={{ padding: 20, borderColor: 'rgba(79,110,247,0.25)' }}
            >
              <div className={styles.channelHeader}>
                <div
                  className={styles.channelIcon}
                  style={{ background: 'var(--blue-dim)' }}
                >
                  <Mail size={14} color="#4f6ef7" />
                </div>
                <div>
                  <div className={styles.channelTitle}>Canal 1 — URL</div>
                  <div className={styles.channelSub}>
                    Envie por e-mail ou chat
                  </div>
                </div>
              </div>
              <div className={`${styles.channelBox} ${styles.channelBoxUrl}`}>
                {url}
              </div>
              <Btn
                onClick={() => {
                  copy(url, () => setCopied1(true));
                  setTimeout(() => setCopied1(false), 2000);
                }}
                variant="secondary"
                size="sm"
                icon={copied1 ? Check : Copy}
                style={{ width: '100%' }}
              >
                {copied1 ? 'Copiado!' : 'Copiar URL'}
              </Btn>
            </Card>

            <Card
              style={{ padding: 20, borderColor: 'rgba(139,92,246,0.25)' }}
            >
              <div className={styles.channelHeader}>
                <div
                  className={styles.channelIcon}
                  style={{ background: 'var(--violet-dim)' }}
                >
                  <Smartphone size={14} color="#8b5cf6" />
                </div>
                <div>
                  <div className={styles.channelTitle}>Canal 2 — Chave</div>
                  <div className={styles.channelSub}>
                    Envie por SMS ou outro app
                  </div>
                </div>
              </div>
              <div className={`${styles.channelBox} ${styles.channelBoxKey}`}>
                {fragment}
              </div>
              <Btn
                onClick={() => {
                  copy(fragment, () => setCopied2(true));
                  setTimeout(() => setCopied2(false), 2000);
                }}
                variant="secondary"
                size="sm"
                icon={copied2 ? Check : Copy}
                style={{ width: '100%' }}
              >
                {copied2 ? 'Copiado!' : 'Copiar Chave'}
              </Btn>
            </Card>
          </div>

          <div className={styles.metaGrid}>
            {[
              {
                icon: Clock,
                label: 'Expira em',
                value: expiryOpts.find((o) => o.v === expiry)?.l,
                color: '#f59e0b',
              },
              {
                icon: Flame,
                label: 'Acesso',
                value: 'Uso único',
                color: '#ef4444',
              },
              {
                icon: ShieldCheck,
                label: 'Cifra',
                value: 'AES-256-GCM',
                color: '#10b981',
              },
            ].map((s, i) => (
              <div key={i} className={styles.metaItem}>
                <s.icon size={15} color={s.color} />
                <div>
                  <div className={styles.metaLabel}>{s.label}</div>
                  <div className={styles.metaValue}>{s.value}</div>
                </div>
              </div>
            ))}
          </div>

          <Btn
            onClick={() => {
              setStep('form');
              setContent('');
              setResult(null);
              setProgress(0);
            }}
            variant="secondary"
            icon={Plus}
            style={{ width: '100%' }}
          >
            Criar Novo Segredo
          </Btn>
        </div>
      )}
    </div>
  );
};

export default CreateScreen;
