import { useState, useCallback, type FC } from 'react';
import {
  BrowserRouter,
  Routes,
  Route,
  Navigate,
  useNavigate,
} from 'react-router-dom';
import { Ban } from 'lucide-react';

import { Auth } from './lib/auth';
import { api } from './lib/api';
import Sidebar from './components/Sidebar';
import Toast from './components/Toast';
import Card from './components/Card';
import type { ToastKind } from './components/Toast';

import CreateScreen from './screens/CreateScreen';
import SecretsScreen from './screens/SecretsScreen';
import ViewSecretScreen from './screens/ViewSecretScreen';
import AdminScreen from './screens/AdminScreen';
import LoginScreen from './screens/LoginScreen';

import styles from './App.module.css';

interface ToastState {
  msg: string;
  kind: ToastKind;
  id: number;
}

/* ── Authenticated shell ───────────────────────────────────────────────── */
const AuthShell: FC<{
  toast: (msg: string, kind?: ToastKind) => void;
}> = ({ toast }) => {
  const navigate = useNavigate();
  const [, setTick] = useState(0); // force re-render on auth change

  const refreshAuth = useCallback(() => {
    setTick((t) => t + 1);
  }, []);

  const onLogout = async () => {
    if (Auth.refresh) {
      await api('/api/v1/auth/logout', {
        method: 'POST',
        body: JSON.stringify({ refresh_token: Auth.refresh }),
      });
    }
    Auth.clear();
    refreshAuth();
    navigate('/login');
  };

  if (!Auth.isAuthed()) {
    return <Navigate to="/login" replace />;
  }

  return (
    <div className={styles.layout}>
      <Sidebar
        isAdmin={Auth.isAdmin()}
        username={Auth.username}
        onLogout={onLogout}
      />
      <main className={styles.main}>
        <Routes>
          <Route path="/" element={<CreateScreen toast={toast} />} />
          <Route
            path="/secrets"
            element={<SecretsScreen toast={toast} />}
          />
          <Route
            path="/admin"
            element={
              Auth.isAdmin() ? (
                <AdminScreen toast={toast} />
              ) : (
                <div className={styles.accessDenied}>
                  <Card style={{ padding: 32, textAlign: 'center', maxWidth: 400 }}>
                    <Ban size={32} color="#ef4444" />
                    <div className={styles.accessDeniedTitle}>
                      Acesso negado
                    </div>
                    <div className={styles.accessDeniedSub}>
                      Apenas administradores podem acessar este painel.
                    </div>
                  </Card>
                </div>
              )
            }
          />
        </Routes>
      </main>
    </div>
  );
};

/* ── Root app ──────────────────────────────────────────────────────────── */
const App: FC = () => {
  const [toastMsg, setToastMsg] = useState<ToastState | null>(null);

  const toast = useCallback(
    (msg: string, kind: ToastKind = 'info') =>
      setToastMsg({ msg, kind, id: Date.now() }),
    [],
  );

  return (
    <BrowserRouter>
      <Routes>
        {/* Public routes */}
        <Route
          path="/s/:slug"
          element={<ViewSecretScreen toast={toast} />}
        />
        <Route
          path="/login"
          element={
            <LoginWrapper toast={toast} />
          }
        />

        {/* Authenticated routes */}
        <Route path="/*" element={<AuthShell toast={toast} />} />
      </Routes>

      {toastMsg && (
        <Toast
          key={toastMsg.id}
          msg={toastMsg.msg}
          kind={toastMsg.kind}
          onDone={() => setToastMsg(null)}
        />
      )}
    </BrowserRouter>
  );
};

/* ── Login wrapper (redirects to / if already authed) ──────────────────── */
const LoginWrapper: FC<{
  toast: (msg: string, kind?: ToastKind) => void;
}> = ({ toast }) => {
  const navigate = useNavigate();
  const [, setTick] = useState(0);

  if (Auth.isAuthed()) {
    return <Navigate to="/" replace />;
  }

  const onAuth = () => {
    setTick((t) => t + 1);
    navigate('/');
  };

  return <LoginScreen onAuth={onAuth} toast={toast} />;
};

export default App;
