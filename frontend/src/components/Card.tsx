import type { CSSProperties, FC, ReactNode } from 'react';
import styles from './Card.module.css';

interface CardProps {
  children: ReactNode;
  style?: CSSProperties;
}

const Card: FC<CardProps> = ({ children, style }) => (
  <div className={styles.card} style={style}>
    {children}
  </div>
);

export default Card;
