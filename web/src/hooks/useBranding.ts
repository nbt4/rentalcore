import { useEffect, useState } from 'react';

export interface BrandingAssets {
  markOnDark: string; markOnLight: string;
  horizontalOnDark: string; horizontalOnLight: string;
  stackedOnDark: string; stackedOnLight: string;
  favicon: string; appIcon: string; maskableIcon: string; print: string;
}

export interface BrandingConfig {
  productName: string; companyName: string; brandName: string;
  assets: BrandingAssets; companyAssets: Partial<BrandingAssets>;
  sidebarLogo: string; loginLogo: string; faviconPath: string;
}

const defaults: BrandingConfig = {
  productName: 'RentalCore', companyName: 'Cores', brandName: '',
  assets: {
    markOnDark: '/static/images/logos/rentalcore_white_icon.svg', markOnLight: '/static/images/logos/rentalcore_black_icon.svg',
    horizontalOnDark: '/static/images/logos/rentalcore_white_side.svg', horizontalOnLight: '/static/images/logos/rentalcore_black_side.svg',
    stackedOnDark: '/static/images/logos/rentalcore_white_full.svg', stackedOnLight: '/static/images/logos/rentalcore_black_full.svg',
    favicon: '/static/images/logos/rentalcore_black_icon.svg', appIcon: '/static/images/app-icons/icon-512.png',
    maskableIcon: '/static/images/app-icons/icon-maskable-512.png', print: '/static/images/logos/rentalcore_black_side.svg',
  },
  companyAssets: {}, sidebarLogo: '/static/images/logos/rentalcore_white_side.svg',
  loginLogo: '/static/images/logos/rentalcore_white_full.svg', faviconPath: '/static/images/logos/rentalcore_black_icon.svg',
};

let cached = defaults;
let started = false;
const listeners = new Set<(value: BrandingConfig) => void>();

function applyDocumentBranding(value: BrandingConfig) {
  const setLink = (selector: string, rel: string, href?: string) => {
    if (!href) return;
    let link = document.querySelector<HTMLLinkElement>(selector);
    if (!link) { link = document.createElement('link'); link.rel = rel; document.head.appendChild(link); }
    link.href = href;
    if (rel === 'icon') link.type = href.toLowerCase().includes('.png') ? 'image/png' : 'image/svg+xml';
  };
  setLink("link[rel~='icon']", 'icon', value.assets.favicon);
  setLink("link[rel='apple-touch-icon']", 'apple-touch-icon', value.assets.appIcon);
}

async function refresh() {
  try {
    const response = await fetch('/api/v1/branding', { cache: 'no-store' });
    if (!response.ok) return;
    const raw = await response.json();
    const assets = { ...defaults.assets, ...(raw.assets || {}) };
    cached = {
      productName: raw.productName || defaults.productName, companyName: raw.companyName || defaults.companyName,
      brandName: raw.brandName || '', assets, companyAssets: raw.companyAssets || {},
      sidebarLogo: assets.horizontalOnDark, loginLogo: assets.stackedOnDark || assets.horizontalOnDark,
      faviconPath: assets.favicon,
    };
    applyDocumentBranding(cached);
    listeners.forEach(listener => listener(cached));
  } catch { /* bundled branding remains available offline */ }
}

function start() {
  if (started) return;
  started = true;
  void refresh();
  window.setInterval(() => void refresh(), 60_000);
}

export function useBranding() {
  const [value, setValue] = useState(cached);
  useEffect(() => { listeners.add(setValue); start(); return () => { listeners.delete(setValue); }; }, []);
  return value;
}
