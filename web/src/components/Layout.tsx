import type { ReactNode } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import {
  Home, Briefcase, Users, BarChart2, FileText,
  Menu, X, LogOut, User, ChevronLeft, ChevronRight, Warehouse, LayoutDashboard,
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
    { path: '/customers', icon: Users, label: 'Kontakte' },
    { path: '/analytics', icon: BarChart2, label: 'Analyse' },
    { path: '/documents', icon: FileText, label: 'Dokumente' },
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
        style={{ height: '60px', borderBottom: '1px solid rgba(255,255,255,0.08)' }}
      >
        <div className="flex items-center justify-between px-4 sm:px-6 h-full">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="p-2 rounded-lg transition-colors cursor-pointer"
              style={{ background: 'none', border: 'none', color: '#A0A0A0' }}
            >
              {!isMobile
                ? (sidebarOpen ? <ChevronLeft className="w-5 h-5" /> : <ChevronRight className="w-5 h-5" />)
                : <Menu className="w-5 h-5" />}
            </button>
            <h1 className="font-bold" style={{ fontSize: '1.125rem' }}>
              <span style={{ color: '#D0021B' }}>Rental</span>
              <span style={{ color: '#ffffff' }}>Core</span>
            </h1>
          </div>
          <div className="text-sm hidden sm:block" style={{ color: '#606060' }}>{companyName}</div>
        </div>
      </header>

      {/* Mobile backdrop */}
      {isMobile && sidebarOpen && (
        <div className="fixed inset-0 bg-black/70 z-40" onClick={close} />
      )}

      {/* Sidebar */}
      <aside className={`fixed left-0 top-0 bottom-0 z-50 glass-dark border-r border-white/10 transition-all duration-300 ease-in-out flex flex-col ${
        isMobile && !sidebarOpen ? '-translate-x-full' : 'translate-x-0'
      } ${isMobile ? 'w-64' : sidebarOpen ? 'w-64' : 'w-20'}`}>

        {/* Mobile header */}
        <div
          className="flex items-center justify-between px-4 py-4 md:hidden"
          style={{ borderBottom: '1px solid rgba(255,255,255,0.08)' }}
        >
          <h2 className="font-bold" style={{ fontSize: '1.125rem' }}>
            <span style={{ color: '#D0021B' }}>Rental</span>
            <span style={{ color: '#ffffff' }}>Core</span>
          </h2>
          <button
            onClick={close}
            className="p-2 rounded-lg cursor-pointer"
            style={{ background: 'none', border: 'none', color: '#A0A0A0' }}
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <nav
          className={`flex-1 overflow-y-auto p-3 space-y-1 ${isMobile ? 'mt-12' : 'mt-[60px]'}`}
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
              background: 'rgba(208,2,27,0.08)',
              color: '#D0021B',
              border: '1px solid rgba(208,2,27,0.15)',
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
                background: isActive(path, exact) ? '#D0021B' : 'transparent',
                color: isActive(path, exact) ? '#ffffff' : '#A0A0A0',
                textDecoration: 'none',
                fontSize: '0.875rem',
                fontWeight: 500,
                boxShadow: isActive(path, exact) ? '0 2px 8px rgba(208,2,27,0.3)' : 'none',
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
          style={{ borderTop: '1px solid rgba(255,255,255,0.08)' }}
        >
          {user && (sidebarOpen || isMobile) && (
            <button
              onClick={() => { close(); navigate('/profile'); }}
              className="flex items-center gap-2.5 px-3 py-2.5 rounded-lg w-full text-left cursor-pointer transition-all"
              style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.06)', color: '#ffffff' }}
            >
              <User className="w-4 h-4 flex-shrink-0" style={{ color: '#D0021B' }} />
              <span className="text-sm font-medium truncate">{user.Username}</span>
            </button>
          )}
          {user && !sidebarOpen && !isMobile && (
            <div
              className="p-2.5 rounded-lg flex justify-center cursor-pointer"
              style={{ background: 'rgba(255,255,255,0.04)' }}
              onClick={() => navigate('/profile')}
            >
              <User className="w-4 h-4" style={{ color: '#D0021B' }} />
            </div>
          )}
          <button
            onClick={handleLogout}
            className={`flex items-center rounded-lg transition-all cursor-pointer ${
              sidebarOpen || isMobile ? 'gap-3 px-3 py-2.5 w-full' : 'justify-center p-3'
            }`}
            style={{ background: 'none', border: 'none', color: '#A0A0A0', fontSize: '0.875rem', fontWeight: 500 }}
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
