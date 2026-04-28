import type { CSSProperties, FC, ReactNode } from 'react';
import type { LucideIcon } from 'lucide-react';
import styles from './Button.module.css';

type BtnVariant = 'primary' | 'secondary' | 'danger' | 'ghost';
type BtnSize = 'sm' | 'md' | 'lg';

interface BtnProps {
  children: ReactNode;
  onClick?: () => void;
  variant?: BtnVariant;
  size?: BtnSize;
  icon?: LucideIcon;
  disabled?: boolean;
  type?: 'button' | 'submit';
  style?: CSSProperties;
}

const Btn: FC<BtnProps> = ({
  children,
  onClick,
  variant = 'primary',
  size = 'md',
  icon: Icon,
  disabled,
  type = 'button',
  style,
}) => {
  const cls = [styles.btn, styles[size], styles[variant]].join(' ');

  return (
    <button
      type={type}
      onClick={disabled ? undefined : onClick}
      disabled={disabled}
      className={cls}
      style={style}
    >
      {Icon && <Icon size={13} />}
      {children}
    </button>
  );
};

export default Btn;
