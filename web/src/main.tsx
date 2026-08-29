import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import './cores-theme.css'
import App from './App'
import { appBasePath, appPath } from './lib/app-paths'

document.addEventListener('wheel', (event) => {
  const target = event.target
  if (target instanceof HTMLInputElement && target.type === 'number' && document.activeElement === target) {
    target.blur()
  }
}, { capture: true, passive: true })

const isStandalone = window.matchMedia('(display-mode: standalone)').matches
  || (navigator as Navigator & { standalone?: boolean }).standalone === true
document.documentElement.classList.toggle('app-standalone', isStandalone)

if (import.meta.env.PROD && 'serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    void navigator.serviceWorker.register(appPath('/sw.js?v=4'), {
      scope: `${appBasePath || ''}/`,
      updateViaCache: 'none',
    }).catch((error: unknown) => {
      console.error('RentalCore service worker registration failed:', error)
    })
  })
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
