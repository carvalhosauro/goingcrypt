import type { FC } from 'react';
import styles from './Spinner.module.css';

interface SpinnerProps {
  size?: number;
  color?: string;
}

const Spinner: FC<SpinnerProps> = ({ size = 20, color = '#4f6ef7' }) => (
  <div
    className={styles.spinner}
    style={{
      width: size,
      height: size,
      border: `2px solid ${color}33`,
      borderTop: `2px solid ${color}`,
    }}
  />
);

export default Spinner;
