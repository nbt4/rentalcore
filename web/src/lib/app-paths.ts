const RENTAL_MOUNT_PATH = '/rental';

export const appBasePath = window.location.pathname === RENTAL_MOUNT_PATH
  || window.location.pathname.startsWith(`${RENTAL_MOUNT_PATH}/`)
  ? RENTAL_MOUNT_PATH
  : '';

export function appPath(path: string): string {
  const normalized = path.startsWith('/') ? path : `/${path}`;
  return `${appBasePath}${normalized}`;
}

export function appAssetPath(value: string): string {
  if (!appBasePath || !value || !value.startsWith('/') || value.startsWith(`${appBasePath}/`)) {
    return value;
  }
  return appPath(value);
}
