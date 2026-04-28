import { useState, useEffect, type FC } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Key,
  Unlock,
  AlertTriangle,
  CheckCircle,
  Eye,
  Copy,
  Flame,
} from 'lucide-react';
import Card from '../components/Card';
import Badge from '../components/Badge';
import Btn from '../components/Button';
import GopherLogo from '../components/GopherLogo';
import { importKeyB64Url, decryptText } from '../lib/crypto';
import { apiJSON } from '../lib/api';
import styles from './ViewSecretScreen.module.css';

interface ViewSecretScreenProps {
  toast: (msg: string, kind?: 'info' | 'error' | 'success') => void;
}

const ViewSecretScreen: FC<ViewSecretScreenProps> = ({ toast }) => {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const [step, setStep] = useState<
    'enter' | 'decrypting' | 'reveal' | 'burned' | 'error'
  >('enter');
  const [keyInput, setKeyInput] = useState('');
  const [showContent, setShowContent] = useState(false);
  const [plaintext, setPlaintext] = useState('');
  const [errMsg, setErrMsg] = useState('');

  // Auto-fill key from URL fragment
  useEffect(() => {
    const hash = window.location.hash;
    if (hash && hash.length > 1) {
      setKeyInput(hash.slice(1));
      history.replaceState({}, '', window.location.pathname);
    }
  }, []);

  const handleDecrypt = async () => {
    const k = keyInput.trim();
    if (!k) return;
    setStep('decrypting');
    setErrMsg('');
    try {
      const res = await apiJSON<{ ciphered_text: string; error?: string }>(
        `/api/v1/links/${encodeURIComponent(slug!)}/access`,
        {
          method: 'POST',
          body: JSON.stringify({ key: k }),
        },
      );
      if (!res.ok) {
        setErrMsg(
          res.body?.error ||
            'Link não encontrado, expirado ou já acessado.',
        );
        setStep('error');
        return;
      }
      const ciphered = res.body!.ciphered_text;
      const aesKey = await importKeyB64Url(k);
      const pt = await decryptText(ciphered, aesKey);
      setPlaintext(pt);
      setStep('reveal');
    } catch {
      setErrMsg('Falha na decriptação — chave incorreta?');
      setStep('error');
    }
  };

  const copyPlain = async () => {
    try {
      await navigator.clipboard.writeText(plaintext);
      toast('Copiado', 'success');
    } catch {
      toast('Falha ao copiar', 'error');
    }
  };

  return (
    <div className={styles.page}>
      <div className={styles.glow1} />
      <div className={styles.glow2} />

      <div className={styles.container}>
        <div className={styles.brand} onClick={() => navigate('/')}>
          <GopherLogo size={48} />
          <div className={styles.brandName}>goingcrypt</div>
        </div>

        {step === 'enter' && (
          <Card style={{ padding: 26 }} >
            <div className={styles.animated}>
              <div className={styles.enterHeader}>
                <div className={styles.enterIcon}>
                  <Key size={17} color="#4f6ef7" />
                </div>
                <div>
                  <div className={styles.enterTitle}>Segredo Protegido</div>
                  <div className={styles.enterSlug}>
                    {window.location.host}/s/{slug}
                  </div>
                </div>
              </div>

              <div className={styles.warningBanner}>
                <AlertTriangle size={13} color="#f59e0b" />
                <span className={styles.warningText}>
                  Este link é de <strong>uso único</strong>. Após a
                  visualização, o conteúdo é destruído permanentemente.
                </span>
              </div>

              <label className={styles.fieldLabel}>
                Chave de Decriptação (Canal 2)
              </label>
              <div className={styles.inputWrap}>
                <div className={styles.inputIcon}>
                  <Key size={14} />
                </div>
                <input
                  value={keyInput}
                  onChange={(e) => setKeyInput(e.target.value)}
                  placeholder="Cole aqui a chave recebida..."
                  className={styles.input}
                  onKeyDown={(e) => e.key === 'Enter' && handleDecrypt()}
                />
              </div>
              <div className={styles.inputHint}>
                A chave é usada para validar o hash e decriptar localmente.
              </div>
              <Btn
                onClick={handleDecrypt}
                variant="primary"
                size="lg"
                icon={Unlock}
                disabled={!keyInput.trim()}
                style={{ width: '100%' }}
              >
                Decriptar e Visualizar
              </Btn>
            </div>
          </Card>
        )}

        {step === 'decrypting' && (
          <Card style={{ padding: 40, textAlign: 'center' }}>
            <div className={styles.animated}>
              <div className={styles.spinnerMd} />
              <div className={styles.decryptingTitle}>
                Decriptando localmente...
              </div>
              <div className={styles.decryptingSub}>
                AES-256-GCM · Web Crypto API
              </div>
            </div>
          </Card>
        )}

        {step === 'reveal' && (
          <Card
            style={{
              padding: 24,
              borderColor: 'rgba(16,185,129,0.2)',
            }}
          >
            <div className={styles.animated}>
              <div className={styles.revealHeader}>
                <div className={styles.revealTitle}>
                  <CheckCircle size={15} color="#10b981" />
                  <span className={styles.revealTitleText}>
                    Conteúdo decriptado
                  </span>
                </div>
                <Badge color="red" dot>
                  Link invalidado
                </Badge>
              </div>

              <div className={styles.contentBox}>
                <div
                  className={`${styles.contentText} ${showContent ? styles.contentVisible : styles.contentHidden}`}
                >
                  {plaintext}
                </div>
                {!showContent && (
                  <div className={styles.revealOverlay}>
                    <Btn
                      onClick={() => setShowContent(true)}
                      variant="secondary"
                      size="sm"
                      icon={Eye}
                    >
                      Revelar conteúdo
                    </Btn>
                  </div>
                )}
              </div>

              {showContent && (
                <div className={styles.actions}>
                  <Btn
                    onClick={copyPlain}
                    variant="secondary"
                    size="sm"
                    icon={Copy}
                    style={{ flex: 1 }}
                  >
                    Copiar
                  </Btn>
                  <Btn
                    onClick={() => setStep('burned')}
                    variant="danger"
                    size="sm"
                    icon={Flame}
                    style={{ flex: 1 }}
                  >
                    Confirmar e Sair
                  </Btn>
                </div>
              )}

              <div className={styles.footnote}>
                Link já foi invalidado no servidor — qualquer acesso futuro
                retornará 404.
              </div>
            </div>
          </Card>
        )}

        {step === 'burned' && (
          <Card style={{ padding: 32, textAlign: 'center' }}>
            <div className={styles.animated}>
              <div
                className={styles.centerIcon}
                style={{
                  background: 'rgba(239,68,68,0.1)',
                  border: '1px solid rgba(239,68,68,0.2)',
                }}
              >
                <Flame size={24} color="#ef4444" />
              </div>
              <div className={styles.centerTitle}>Segredo destruído</div>
              <div className={styles.centerSub}>
                Link invalidado. Qualquer acesso futuro retornará 404.
              </div>
              <Badge color="gray">
                {window.location.host}/s/{slug}
              </Badge>
            </div>
          </Card>
        )}

        {step === 'error' && (
          <Card
            style={{
              padding: 32,
              textAlign: 'center',
              borderColor: 'rgba(239,68,68,0.25)',
            }}
          >
            <div className={styles.animated}>
              <div
                className={styles.centerIcon}
                style={{
                  background: 'rgba(239,68,68,0.1)',
                  border: '1px solid rgba(239,68,68,0.2)',
                }}
              >
                <AlertTriangle size={24} color="#ef4444" />
              </div>
              <div className={styles.centerTitle}>
                Não foi possível abrir
              </div>
              <div className={styles.centerSub}>{errMsg}</div>
              <Btn
                onClick={() => {
                  setStep('enter');
                  setErrMsg('');
                }}
                variant="secondary"
                size="sm"
              >
                Tentar novamente
              </Btn>
            </div>
          </Card>
        )}
      </div>
    </div>
  );
};

export default ViewSecretScreen;
