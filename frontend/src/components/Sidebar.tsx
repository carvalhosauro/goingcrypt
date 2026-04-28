import type { FC } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { Plus, List, Shield, LogOut } from 'lucide-react';
import GopherLogo from './GopherLogo';
import styles from './Sidebar.module.css';

interface SidebarProps {
  isAdmin: boolean;
  username: string;
  onLogout: () => void;
}

const navItems = [
  { id: 'create',  label: 'Novo Segredo',  icon: Plus,   path: '/' },
  { id: 'secrets', label: 'Meus Segredos', icon: List,   path: '/secrets' },
];

const adminItem = { id: 'admin', label: 'Admin', icon: Shield, path: '/admin' };

function routeName(pathname: string): string {
  if (pathname === '/secrets') return 'secrets';
  if (pathname === '/admin') return 'admin';
  return 'create';
}

const Sidebar: FC<SidebarProps> = ({ isAdmin, username, onLogout }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const current = routeName(location.pathname);
  const initials = (username || '?').slice(0, 2).toUpperCase();

  const items = isAdmin ? [...navItems, adminItem] : navItems;

  return (
    <aside className={styles.sidebar}>
      <div className={styles.brand} onClick={() => navigate('/')}>
        <GopherLogo size={36} />
        <div>
          <div className={styles.brandName}>goingcrypt</div>
          <div className={styles.brandSub}>secret sharing</div>
        </div>
      </div>

      <nav className={styles.nav}>
        {items.map((item) => {
          const active = current === item.id;
          const IconComp = item.icon;
          return (
            <button
              key={item.id}
              onClick={() => navigate(item.path)}
              className={`${styles.navItem} ${active ? styles.active : ''}`}
            >
              <IconComp size={15} color={active ? '#4f6ef7' : '#4a5568'} />
              {item.label}
              {item.id === 'admin' && (
                <span className={styles.adminTag}>ADMIN</span>
              )}
            </button>
          );
        })}
      </nav>

      <div className={styles.footer}>
        <button className={styles.userBtn} onClick={onLogout}>
          <div className={styles.avatar}>{initials}</div>
          <div className={styles.userInfo}>
            <div className={styles.userName}>{username || 'usuário'}</div>
            <div className={styles.userRole}>
              {isAdmin ? 'admin' : 'user'} · sair
            </div>
          </div>
          <LogOut size={13} color="#4a5568" />
        </button>
      </div>
    </aside>
  );
};

export default Sidebar;
