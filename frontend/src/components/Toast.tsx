import { useEffect, type FC } from 'react';
import styles from './Toast.module.css';

export type ToastKind = 'info' | 'error' | 'success';

interface ToastProps {
  msg: string;
  kind?: ToastKind;
  onDone: () => void;
}

const kindColors: Record<ToastKind, string> = {
  info: '#4f6ef7',
  error: '#ef4444',
  success: '#10b981',
};

const Toast: FC<ToastProps> = ({ msg, kind = 'info', onDone }) => {
  useEffect(() => {
    const t = setTimeout(onDone, 3500);
    return () => clearTimeout(t);
  }, [onDone]);

  const c = kindColors[kind];

  return (
    <div
      className={styles.toast}
      style={{
        border: `1px solid ${c}55`,
        borderLeft: `3px solid ${c}`,
      }}
    >
      {msg}
    </div>
  );
};

export default Toast;
