import { useState, type FC, type FormEvent } from 'react';
import { User, Lock, Eye, EyeOff } from 'lucide-react';
import Card from '../components/Card';
import GopherLogo from '../components/GopherLogo';
import { Auth } from '../lib/auth';
import { apiJSON } from '../lib/api';
import styles from './LoginScreen.module.css';

interface LoginScreenProps {
  onAuth: () => void;
  toast: (msg: string, kind?: 'info' | 'error' | 'success') => void;
}

const LoginScreen: FC<LoginScreenProps> = ({ onAuth, toast }) => {
  const [mode, setMode] = useState<'login' | 'signup'>('login');
  const [username, setUsername] = useState('');
  const [pass, setPass] = useState('');
  const [showPass, setShowPass] = useState(false);
  const [loading, setLoading] = useState(false);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (!username || !pass) return;
    setLoading(true);
    const path =
      mode === 'login' ? '/api/v1/auth/login' : '/api/v1/auth/signup';
    const res = await apiJSON<{
      access_token: string;
      refresh_token: string;
      error?: string;
    }>(path, {
      method: 'POST',
      body: JSON.stringify({ username, password: pass, device_name: 'web' }),
    });
    setLoading(false);
    if (!res.ok) {
      const msg =
        res.body?.error ||
        (res.status === 401
          ? 'Credenciais inválidas'
          : res.status === 409
            ? 'Usuário já existe'
            : 'Falha');
      toast(msg, 'error');
      return;
    }
    Auth.set(res.body!.access_token, res.body!.refresh_token);
    Auth.setUsername(username);
    onAuth();
  };

  return (
    <div className={styles.page}>
      <div className={styles.glow1} />
      <div className={styles.glow2} />

      <div className={styles.container}>
        <div className={styles.brand}>
          <GopherLogo size={52} />
          <div className={styles.brandName}>goingcrypt</div>
          <div className={styles.brandSub}>
            Compartilhamento seguro de segredos
          </div>
        </div>

        <Card style={{ padding: 26 }}>
          <div className={styles.animated}>
            <div className={styles.tabSwitcher}>
              {(['login', 'signup'] as const).map((m) => (
                <button
                  key={m}
                  onClick={() => setMode(m)}
                  className={`${styles.tabBtn} ${mode === m ? styles.active : ''}`}
                >
                  {m === 'login' ? 'Entrar' : 'Criar Conta'}
                </button>
              ))}
            </div>

            <form onSubmit={submit}>
              <div className={styles.fieldGroup}>
                <label className={styles.fieldLabel}>Usuário</label>
                <div className={styles.inputWrap}>
                  <div className={styles.inputIcon}>
                    <User size={14} />
                  </div>
                  <input
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    placeholder="seu-usuario"
                    autoComplete="username"
                    className={styles.input}
                  />
                </div>
              </div>

              <div className={styles.fieldGroupLast}>
                <label className={styles.fieldLabel}>Senha</label>
                <div className={styles.inputWrap}>
                  <div className={styles.inputIcon}>
                    <Lock size={14} />
                  </div>
                  <input
                    type={showPass ? 'text' : 'password'}
                    value={pass}
                    onChange={(e) => setPass(e.target.value)}
                    placeholder="••••••••"
                    autoComplete={
                      mode === 'login'
                        ? 'current-password'
                        : 'new-password'
                    }
                    className={`${styles.input} ${styles.inputPassword}`}
                  />
                  <button
                    type="button"
                    onClick={() => setShowPass((v) => !v)}
                    className={styles.togglePass}
                  >
                    {showPass ? (
                      <EyeOff size={15} />
                    ) : (
                      <Eye size={15} />
                    )}
                  </button>
                </div>
                {mode === 'signup' && (
                  <div className={styles.passHint}>
                    Mínimo 8 caracteres.
                  </div>
                )}
              </div>

              <button
                type="submit"
                disabled={loading}
                className={styles.submitBtn}
              >
                {loading
                  ? 'Verificando...'
                  : mode === 'login'
                    ? 'Entrar →'
                    : 'Criar Conta →'}
              </button>
            </form>
          </div>
        </Card>

        <div className={styles.terms}>
          Ao continuar, você concorda com os{' '}
          <span className={styles.termsLink}>Termos de Uso</span>
        </div>
      </div>
    </div>
  );
};

export default LoginScreen;
