import type { ReactNode } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import {
  Home, Briefcase, Users, BarChart2, FileText, Settings,
  Menu, X, LogOut, User, ChevronLeft, ChevronRight, Warehouse,
} from 'lucide-react';
import { useState, useEffect, useMemo } from 'react';
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

  const userRoles = (user?.Roles ?? user?.roles ?? []) as { name?: string; Name?: string }[];
  const isAdmin = useMemo(() =>
    userRoles.some((r) => ['admin', 'manager'].includes((r?.name || r?.Name || '').toLowerCase())),
    [userRoles]
  );

  const navItems = [
    { path: '/', icon: Home, label: 'Dashboard', exact: true },
    { path: '/jobs', icon: Briefcase, label: 'Jobs' },
    { path: '/customers', icon: Users, label: 'Kunden' },
    { path: '/analytics', icon: BarChart2, label: 'Analyse' },
    { path: '/documents', icon: FileText, label: 'Dokumente' },
  ];

  const isActive = (path: string, exact = false) =>
    exact ? location.pathname === path : location.pathname.startsWith(path);

  return (
    <div className="min-h-screen bg-dark">
      {/* Header */}
      <header className={`fixed top-0 right-0 z-50 glass-dark border-b border-white/10 transition-all duration-300 ${
        !isMobile && sidebarOpen ? 'left-64' : !isMobile ? 'left-20' : 'left-0'
      }`}>
        <div className="flex items-center justify-between px-3 sm:px-6 py-3 sm:py-4">
          <div className="flex items-center gap-2 sm:gap-4">
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="p-2 hover:bg-white/10 rounded-lg transition-colors"
            >
              {!isMobile
                ? (sidebarOpen ? <ChevronLeft className="w-5 h-5" /> : <ChevronRight className="w-5 h-5" />)
                : <Menu className="w-5 h-5" />}
            </button>
            <h1 className="text-lg sm:text-2xl font-bold">
              <span className="text-accent-red">Rental</span>
              <span className="text-white">Core</span>
            </h1>
          </div>
          <div className="text-xs sm:text-sm text-gray-400 hidden sm:block">{companyName}</div>
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
        <div className="flex items-center justify-between px-4 py-4 border-b border-white/10 md:hidden">
          <h2 className="text-lg font-bold">
            <span className="text-accent-red">Rental</span>
            <span className="text-white">Core</span>
          </h2>
          <button onClick={close} className="p-2 hover:bg-white/10 rounded-lg transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        <nav className={`flex-1 overflow-y-auto p-4 space-y-2 ${isMobile ? 'mt-12' : 'mt-20'}`}>
          {/* Cross-nav to WarehouseCore */}
          <a
            href={warehouseURL}
            className={`flex items-center rounded-lg transition-all bg-accent-red/10 text-accent-red hover:bg-accent-red hover:text-white border border-accent-red/20 ${
              sidebarOpen || isMobile ? 'gap-3 px-4 py-3' : 'justify-center p-3'
            }`}
            title="Zu WarehouseCore wechseln"
          >
            <Warehouse className="w-5 h-5 flex-shrink-0" />
            {(sidebarOpen || isMobile) && <span className="font-semibold">WarehouseCore</span>}
          </a>

          {navItems.map(({ path, icon: Icon, label, exact }) => (
            <Link
              key={path}
              to={path}
              onClick={close}
              className={`flex items-center rounded-lg transition-all ${
                isActive(path, exact)
                  ? 'bg-accent-red text-white shadow-lg shadow-accent-red/20'
                  : 'text-gray-400 hover:bg-white/10 hover:text-white'
              } ${sidebarOpen || isMobile ? 'gap-3 px-4 py-3' : 'justify-center p-3'}`}
              title={!sidebarOpen && !isMobile ? label : ''}
            >
              <Icon className="w-5 h-5 flex-shrink-0" />
              {(sidebarOpen || isMobile) && <span className="font-medium">{label}</span>}
            </Link>
          ))}

          {isAdmin && (
            <Link
              to="/admin"
              onClick={close}
              className={`flex items-center rounded-lg transition-all ${
                location.pathname.startsWith('/admin')
                  ? 'bg-accent-red text-white shadow-lg shadow-accent-red/20'
                  : 'text-gray-400 hover:bg-white/10 hover:text-white'
              } ${sidebarOpen || isMobile ? 'gap-3 px-4 py-3' : 'justify-center p-3'}`}
              title={!sidebarOpen && !isMobile ? 'Admin' : ''}
            >
              <Settings className="w-5 h-5 flex-shrink-0" />
              {(sidebarOpen || isMobile) && <span className="font-medium">Admin</span>}
            </Link>
          )}
        </nav>

        {/* User / Logout */}
        <div className={`p-4 border-t border-white/10 bg-dark/50 ${!sidebarOpen && !isMobile ? 'flex flex-col items-center' : ''}`}>
          {user && (sidebarOpen || isMobile) && (
            <button
              onClick={() => { close(); navigate('/profile'); }}
              className="mb-3 px-4 py-2 rounded-lg bg-white/5 text-left w-full hover:bg-white/10"
            >
              <div className="flex items-center gap-2 text-sm">
                <User className="w-4 h-4 text-accent-red" />
                <span className="text-gray-300 font-medium underline underline-offset-2">{user.Username}</span>
              </div>
            </button>
          )}
          {user && !sidebarOpen && !isMobile && (
            <div className="mb-3 p-2 rounded-lg bg-white/5 flex justify-center">
              <User className="w-5 h-5 text-accent-red" />
            </div>
          )}
          <button
            onClick={handleLogout}
            className={`flex items-center rounded-lg transition-all text-gray-400 hover:bg-red-500/10 hover:text-red-400 ${
              sidebarOpen || isMobile ? 'gap-3 px-4 py-3 w-full' : 'justify-center p-3'
            }`}
          >
            <LogOut className="w-5 h-5 flex-shrink-0" />
            {(sidebarOpen || isMobile) && <span className="font-medium">Abmelden</span>}
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className={`pt-14 sm:pt-16 transition-all duration-300 ${isMobile ? 'ml-0' : sidebarOpen ? 'ml-64' : 'ml-20'}`}>
        <div className="p-3 sm:p-6">{children}</div>
      </main>
    </div>
  );
}
