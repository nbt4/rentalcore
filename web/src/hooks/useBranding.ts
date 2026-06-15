import { useState, useEffect } from 'react';

export interface BrandingConfig {
  companyName: string;
  brandName: string;
  sidebarLogo: string;
  loginLogo: string;
  faviconPath: string;
  logoSizeSidebar: number;
  logoSizeLogin: number;
}

function readInitial(): BrandingConfig {
  const app = (window as any).__APP_CONFIG__;
  const b = app?.branding;
  return {
    companyName: b?.companyName || app?.companyName || 'RentalCore',
    brandName: b?.brandName || '',
    sidebarLogo: b?.sidebarLogo || '/static/images/logos/rentalcore_white_side.svg',
    loginLogo: b?.loginLogo || '/static/images/logos/rentalcore_white_side.svg',
    faviconPath: b?.favicon || '',
    logoSizeSidebar: b?.logoSizeSidebar || 100,
    logoSizeLogin: b?.logoSizeLogin || 100,
  };
}

let cached = readInitial();

export function useBranding() {
  const [branding, setBranding] = useState<BrandingConfig>(cached);

  useEffect(() => {
    let active = true;

    const poll = async () => {
      try {
        const res = await fetch('/api/v1/branding');
        if (!res.ok || !active) return;
        const raw = await res.json();
        // Apply same fallbacks as readInitial
        const data: BrandingConfig = {
          companyName: raw.companyName || readInitial().companyName,
          brandName: raw.brandName || '',
          sidebarLogo: raw.sidebarLogo || '/static/images/logos/rentalcore_white_side.svg',
          loginLogo: raw.loginLogo || '/static/images/logos/rentalcore_white_side.svg',
          faviconPath: raw.faviconPath || '',
          logoSizeSidebar: raw.logoSizeSidebar || 100,
          logoSizeLogin: raw.logoSizeLogin || 100,
        };
        cached = data;
        if (active) setBranding(data);
      } catch { /* network error — keep current */ }
    };

    // Poll every 2 seconds for near-instant updates
    const interval = setInterval(poll, 2000);
    return () => { active = false; clearInterval(interval); };
  }, []);

  return branding;
}
