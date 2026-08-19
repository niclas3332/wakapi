import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './styles/index.css'
import { serverConfig } from './lib/bootstrap'

const el = document.getElementById('app')
if (!el) throw new Error('wakapi: #app mount point missing')

createRoot(el).render(
  <StrictMode>
    <main className="mx-auto max-w-3xl p-8">
      <h1 className="text-2xl font-semibold">Wakapi</h1>
      <p className="mt-2 text-slate-400">
        Frontend shell is running. Version {serverConfig.version}, base path{' '}
        <code>{serverConfig.basePath || '/'}</code>.
      </p>
    </main>
  </StrictMode>,
)
