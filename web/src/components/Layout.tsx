import type { ReactNode } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import {
  Home, Briefcase, BarChart2, FileText,
  Menu, X, LogOut, User, ChevronLeft, ChevronRight, LayoutDashboard,
  Users, Star, MapPin,
} from 'lucide-react';
import { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useBranding } from '../hooks/useBranding';
import { suiteGreetingName } from '../lib/cores-design';
import { appBasePath } from '../lib/app-paths';

interface LayoutProps { children: ReactNode }

export function Layout({ children }: LayoutProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const branding = useBranding();
  const companyName = branding.companyName;

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

  useEffect(() => {
    if (isMobile) setSidebarOpen(false);
  }, [location.pathname, isMobile]);

  useEffect(() => {
    const locked = isMobile && sidebarOpen;
    document.documentElement.classList.toggle('modal-open', locked);
    document.body.classList.toggle('modal-open', locked);
    return () => {
      document.documentElement.classList.remove('modal-open');
      document.body.classList.remove('modal-open');
    };
  }, [isMobile, sidebarOpen]);

  const close = () => { if (isMobile) setSidebarOpen(false); };

  const handleLogout = async () => { await logout(); navigate('/login'); };

  const getCoresDashboardURL = () => {
    if (appBasePath) return `${window.location.origin}/`;
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
    <div className="mobile-app-shell min-h-screen bg-dark">
      {/* Header */}
      <header
        className="mobile-app-header md:hidden fixed top-0 right-0 left-0 z-50 glass-dark"
        style={{ height: 'var(--app-header-height)', borderBottom: '1px solid var(--border-subtle)' }}
      >
        <div className="flex items-center justify-between px-4 sm:px-6 h-full">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="p-2 rounded-lg transition-colors cursor-pointer"
              style={{ background: 'none', border: 'none', color: 'var(--text-secondary)' }}
              aria-label={sidebarOpen ? 'Navigation schließen' : 'Navigation öffnen'}
            >
              {!isMobile
                ? (sidebarOpen ? <ChevronLeft className="w-5 h-5" /> : <ChevronRight className="w-5 h-5" />)
                : <Menu className="w-5 h-5" />}
            </button>
          </div>
          <div className="text-sm hidden sm:block" style={{ color: 'var(--text-tertiary)' }}>{companyName}</div>
        </div>
      </header>

      {/* Mobile backdrop */}
      {isMobile && sidebarOpen && (
        <button className="mobile-app-backdrop fixed inset-0 z-40" style={{ background: 'var(--bg-overlay)' }} onClick={close} aria-label="Navigation schließen" />
      )}

      {/* Sidebar */}
      <aside className={`mobile-app-drawer fixed left-0 top-0 bottom-0 z-50 glass-dark transition-all duration-300 ease-in-out flex flex-col ${
        isMobile && !sidebarOpen ? '-translate-x-full' : 'translate-x-0'
      } ${isMobile ? 'w-64' : sidebarOpen ? 'w-64' : 'w-20'}`}
        style={{ borderRight: '1px solid var(--border-subtle)' }}>

        <button
          type="button"
          onClick={() => setSidebarOpen((open) => !open)}
          className="hidden md:flex absolute -right-3 top-[68px] z-10 h-6 w-6 items-center justify-center rounded-full"
          style={{ background: 'var(--surface-3)', border: '1px solid var(--border-default)', color: 'var(--text-secondary)' }}
          aria-label={sidebarOpen ? 'Sidebar zuklappen' : 'Sidebar aufklappen'}
          aria-expanded={sidebarOpen}
        >
          {sidebarOpen ? <ChevronLeft className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
        </button>

        {/* Sidebar header (mobile + desktop) */}
        <div
          className="h-20 flex items-center justify-center px-2 overflow-hidden"
          style={{ borderBottom: '1px solid var(--border-subtle)' }}
        >
          <img
            src={sidebarOpen || isMobile
              ? branding.assets.horizontalOnDark
              : branding.assets.markOnDark}
            alt={branding.productName}
            className={sidebarOpen || isMobile ? 'h-12 w-44 flex-shrink-0 object-contain' : 'h-10 w-10 flex-shrink-0 object-contain'}
          />
          {isMobile && (
            <button
              onClick={close}
              className="absolute right-2 p-2 rounded-lg cursor-pointer"
              style={{ background: 'none', border: 'none', color: 'var(--text-secondary)' }}
              aria-label="Navigation schließen"
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
            <div
              className="flex items-center gap-2.5 px-3 py-2.5 rounded-lg w-full text-left"
              style={{ background: 'var(--bg-subtle)', border: '1px solid var(--border-subtle)', color: 'var(--text-primary)' }}
            >
              <User className="w-4 h-4 flex-shrink-0" style={{ color: 'var(--accent-red)' }} />
              <span className="text-sm font-medium truncate">{suiteGreetingName(user)}</span>
            </div>
          )}
          {user && !sidebarOpen && !isMobile && (
            <div
              className="p-2.5 rounded-lg flex justify-center"
              style={{ background: 'var(--bg-subtle)' }}
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
        className={`mobile-app-main transition-all duration-300 ${isMobile ? 'ml-0' : sidebarOpen ? 'ml-64' : 'ml-20'}`}
        style={{ paddingTop: isMobile ? 'var(--app-header-height)' : 0 }}
      >
        <div className="mobile-app-content suite-page p-4 sm:p-6">{children}</div>
      </main>

      <nav className="mobile-app-tabbar" aria-label="Hauptnavigation">
        {[
          { path: '/', icon: Home, label: 'Start', exact: true },
          { path: '/jobs', icon: Briefcase, label: 'Jobs' },
          { path: '/analytics', icon: BarChart2, label: 'Analyse' },
        ].map(({ path, icon: Icon, label, exact }) => (
          <Link key={path} to={path} className={`mobile-app-tab ${isActive(path, exact) ? 'is-active' : ''}`}>
            <Icon aria-hidden="true" />
            <span>{label}</span>
          </Link>
        ))}
        <button type="button" className={`mobile-app-tab ${sidebarOpen ? 'is-active' : ''}`} onClick={() => setSidebarOpen(true)}>
          <Menu aria-hidden="true" />
          <span>Mehr</span>
        </button>
      </nav>
    </div>
  );
}
