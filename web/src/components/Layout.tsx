import type { ReactNode } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import {
  Home, Briefcase, BarChart2, FileText,
  Menu, X, LogOut, User, ChevronLeft, ChevronRight, Warehouse, LayoutDashboard,
  Users, Star, MapPin,
} from 'lucide-react';
import { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';

interface LayoutProps { children: ReactNode }

export function Layout({ children }: LayoutProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const companyName = (window as unknown as { __APP_CONFIG__?: { companyName?: string } }).__APP_CONFIG__?.companyName || 'Tsunami Events';

  useEffect(() => {
    const check = () => {
      const mobile = window.innerWidth < 768;
      setIsMobile(mobile);
      setSidebarOpen(!mobile);
    };
    check();
    window.addEventListener('resize', check);
    return () => window.removeEventListener('resize', check);
  }, []);

  const close = () => { if (isMobile) setSidebarOpen(false); };

  const handleLogout = async () => { await logout(); navigate('/login'); };

  const getWarehouseCoreURL = () => {
    const wDomain = (window as unknown as { __APP_CONFIG__?: { warehouseCoreDomain?: string } }).__APP_CONFIG__?.warehouseCoreDomain;
    const proto = window.location.protocol;
    if (wDomain && wDomain !== '') return `${proto}//${wDomain}`;
    const { hostname, port } = window.location;
    if (hostname.startsWith('rent.')) return `${proto}//${hostname.replace(/^rent\./, 'warehouse.')}`;
    if (port === '8081') return `${proto}//${hostname}:8082`;
    return `${proto}//${hostname}:8082`;
  };

  const warehouseURL = getWarehouseCoreURL();

  const getCoresDashboardURL = () => {
    const { hostname, port, protocol } = window.location;
    if (port === '8081') return `${protocol}//${hostname}:8080`;
    if (hostname.startsWith('rent.')) return `${protocol}//${hostname.replace(/^rent\./, 'cores.')}`;
    return `${protocol}//${hostname}:8080`;
  };
  const dashboardURL = getCoresDashboardURL();

  const navItems = [
    { path: '/', icon: Home, label: 'Dashboard', exact: true },
    { path: '/jobs', icon: Briefcase, label: 'Jobs' },
    { path: '/analytics', icon: BarChart2, label: 'Analyse' },
    { path: '/documents', icon: FileText, label: 'Dokumente' },
    { path: '/employees', icon: Users, label: 'Mitarbeiter' },
    { path: '/venues', icon: MapPin, label: 'Veranstaltungsorte' },
    { path: '/admin/skills', icon: Star, label: 'Skills' },
  ];

  const isActive = (path: string, exact = false) =>
    exact ? location.pathname === path : location.pathname.startsWith(path);

  return (
    <div className="min-h-screen bg-dark">
      {/* Header */}
      <header
        className={`fixed top-0 right-0 z-50 glass-dark transition-all duration-300 ${
          !isMobile && sidebarOpen ? 'left-64' : !isMobile ? 'left-20' : 'left-0'
        }`}
        style={{ height: '60px', borderBottom: '1px solid var(--border-subtle)' }}
      >
        <div className="flex items-center justify-between px-4 sm:px-6 h-full">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="p-2 rounded-lg transition-colors cursor-pointer"
              style={{ background: 'none', border: 'none', color: 'var(--text-secondary)' }}
            >
              {!isMobile
                ? (sidebarOpen ? <ChevronLeft className="w-5 h-5" /> : <ChevronRight className="w-5 h-5" />)
                : <Menu className="w-5 h-5" />}
            </button>
            <img
              src="/static/images/logos/rentalcore_white_side.svg"
              alt="RentalCore"
              className="h-12"
            />
          </div>
          <div className="text-sm hidden sm:block" style={{ color: 'var(--text-tertiary)' }}>{companyName}</div>
        </div>
      </header>

      {/* Mobile backdrop */}
      {isMobile && sidebarOpen && (
        <div className="fixed inset-0 z-40" style={{ background: 'var(--bg-overlay)' }} onClick={close} />
      )}

      {/* Sidebar */}
      <aside className={`fixed left-0 top-0 bottom-0 z-50 glass-dark transition-all duration-300 ease-in-out flex flex-col ${
        isMobile && !sidebarOpen ? '-translate-x-full' : 'translate-x-0'
      } ${isMobile ? 'w-64' : sidebarOpen ? 'w-64' : 'w-20'}`}
        style={{ borderRight: '1px solid var(--border-subtle)' }}>

        {/* Sidebar header (mobile + desktop) */}
        <div
          className="flex items-center justify-between px-4 py-4"
          style={{ borderBottom: '1px solid var(--border-subtle)' }}
        >
          <img
            src={sidebarOpen || isMobile
              ? '/static/images/logos/rentalcore_white_side.svg'
              : '/static/images/logos/rentalcore_white_icon.svg'}
            alt="RentalCore"
            className={sidebarOpen || isMobile ? 'h-12' : 'h-14 mx-auto'}
          />
          {isMobile && (
            <button
              onClick={close}
              className="p-2 rounded-lg cursor-pointer"
              style={{ background: 'none', border: 'none', color: 'var(--text-secondary)' }}
            >
              <X className="w-5 h-5" />
            </button>
          )}
        </div>

        <nav
          className="flex-1 overflow-y-auto p-3 space-y-1"
          style={{ scrollbarWidth: 'none' }}
        >
          {/* Cores Dashboard link */}
          <a
            href={dashboardURL}
            className="flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-semibold text-accent-red hover:bg-accent-red/10 transition-colors mb-2"
          >
            <LayoutDashboard className="w-4 h-4 flex-shrink-0" />
            {(sidebarOpen || isMobile) && <span>← Cores</span>}
          </a>

          {/* Cross-nav to WarehouseCore */}
          <a
            href={warehouseURL}
            className={`flex items-center rounded-lg transition-all ${
              sidebarOpen || isMobile ? 'gap-3 px-3 py-2.5' : 'justify-center p-3'
            }`}
            style={{
              background: 'rgba(var(--accent-red-rgb), 0.08)',
              color: 'var(--accent-red)',
              border: '1px solid rgba(var(--accent-red-rgb), 0.15)',
              textDecoration: 'none',
              fontSize: '0.875rem',
              fontWeight: 600,
            }}
            title="Zu WarehouseCore wechseln"
          >
            <Warehouse className="w-4 h-4 flex-shrink-0" />
            {(sidebarOpen || isMobile) && <span>WarehouseCore</span>}
          </a>

          <div style={{ height: '4px' }} />

          {navItems.map(({ path, icon: Icon, label, exact }) => (
            <Link
              key={path}
              to={path}
              onClick={close}
              className={`flex items-center rounded-lg transition-all ${
                sidebarOpen || isMobile ? 'gap-3 px-3 py-2.5' : 'justify-center p-3'
              }`}
              style={{
                background: isActive(path, exact) ? 'var(--accent-red)' : 'transparent',
                color: isActive(path, exact) ? 'var(--text-primary)' : 'var(--text-secondary)',
                textDecoration: 'none',
                fontSize: '0.875rem',
                fontWeight: 500,
                boxShadow: isActive(path, exact) ? 'var(--shadow-accent)' : 'none',
              }}
              title={!sidebarOpen && !isMobile ? label : ''}
            >
              <Icon className="w-4 h-4 flex-shrink-0" />
              {(sidebarOpen || isMobile) && <span>{label}</span>}
            </Link>
          ))}

        </nav>

        {/* User / Logout */}
        <div
          className={`p-3 flex flex-col gap-1 ${!sidebarOpen && !isMobile ? 'items-center' : ''}`}
          style={{ borderTop: '1px solid var(--border-subtle)' }}
        >
          {user && (sidebarOpen || isMobile) && (
            <button
              onClick={() => { close(); navigate('/profile'); }}
              className="flex items-center gap-2.5 px-3 py-2.5 rounded-lg w-full text-left cursor-pointer transition-all"
              style={{ background: 'var(--bg-subtle)', border: '1px solid var(--border-subtle)', color: 'var(--text-primary)' }}
            >
              <User className="w-4 h-4 flex-shrink-0" style={{ color: 'var(--accent-red)' }} />
              <span className="text-sm font-medium truncate">{user.Username}</span>
            </button>
          )}
          {user && !sidebarOpen && !isMobile && (
            <div
              className="p-2.5 rounded-lg flex justify-center cursor-pointer"
              style={{ background: 'var(--bg-subtle)' }}
              onClick={() => navigate('/profile')}
            >
              <User className="w-4 h-4" style={{ color: 'var(--accent-red)' }} />
            </div>
          )}
          <button
            onClick={handleLogout}
            className={`flex items-center rounded-lg transition-all cursor-pointer ${
              sidebarOpen || isMobile ? 'gap-3 px-3 py-2.5 w-full' : 'justify-center p-3'
            }`}
            style={{ background: 'none', border: 'none', color: 'var(--text-secondary)', fontSize: '0.875rem', fontWeight: 500 }}
          >
            <LogOut className="w-4 h-4 flex-shrink-0" />
            {(sidebarOpen || isMobile) && <span>Abmelden</span>}
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main
        className={`transition-all duration-300 ${isMobile ? 'ml-0' : sidebarOpen ? 'ml-64' : 'ml-20'}`}
        style={{ paddingTop: '60px' }}
      >
        <div className="p-4 sm:p-6">{children}</div>
      </main>
    </div>
  );
}
