import { useEffect } from 'react';
import { centralLoginURL } from '../lib/suite-auth';

export function CentralLoginRedirect() {
  useEffect(() => {
    window.location.replace(centralLoginURL());
  }, []);

  return (
    <div className="suite-auth-page" role="status" aria-live="polite">
      <p style={{ color: 'var(--text-secondary)' }}>Weiter zur Cores-Anmeldung …</p>
    </div>
  );
}
