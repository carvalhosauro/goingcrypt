import type { FC, ReactNode } from 'react';
import styles from './Badge.module.css';

type BadgeColor = 'blue' | 'green' | 'red' | 'amber' | 'violet' | 'gray';

interface BadgeProps {
  children: ReactNode;
  color?: BadgeColor;
  dot?: boolean;
}

const palette: Record<BadgeColor, { bg: string; text: string; bd: string }> = {
  blue:   { bg: 'rgba(79,110,247,0.12)',  text: '#818cf8', bd: 'rgba(79,110,247,0.2)' },
  green:  { bg: 'rgba(16,185,129,0.1)',   text: '#34d399', bd: 'rgba(16,185,129,0.18)' },
  red:    { bg: 'rgba(239,68,68,0.1)',    text: '#f87171', bd: 'rgba(239,68,68,0.18)' },
  amber:  { bg: 'rgba(245,158,11,0.1)',   text: '#fbbf24', bd: 'rgba(245,158,11,0.18)' },
  violet: { bg: 'rgba(139,92,246,0.12)',  text: '#a78bfa', bd: 'rgba(139,92,246,0.2)' },
  gray:   { bg: 'rgba(100,116,139,0.12)', text: '#94a3b8', bd: 'rgba(100,116,139,0.18)' },
};

const Badge: FC<BadgeProps> = ({ children, color = 'blue', dot = false }) => {
  const c = palette[color];
  return (
    <span
      className={styles.badge}
      style={{ background: c.bg, color: c.text, border: `1px solid ${c.bd}` }}
    >
      {dot && <span className={styles.dot} style={{ background: c.text }} />}
      {children}
    </span>
  );
};

export default Badge;
