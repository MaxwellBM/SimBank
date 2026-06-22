import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import client from '../api/client'
import ChatWidget from '../components/ChatWidget'

function formatCurrency(n) {
  return '$' + Number(n).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function formatTime(ts) {
  return new Date(ts).toLocaleDateString('es-ES', {
    day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit',
  })
}

function formatDate() {
  return new Date().toLocaleDateString('es-ES', {
    weekday: 'long', day: 'numeric', month: 'long', year: 'numeric',
  })
}

function TxIcon({ type }) {
  const cfg = type === 'deposit'
    ? { bg: 'bg-emerald-500/15', fg: 'text-emerald-400', icon: '+' }
    : type === 'withdrawal'
    ? { bg: 'bg-red-500/15', fg: 'text-red-400', icon: '−' }
    : { bg: 'bg-cyan-500/15', fg: 'text-cyan-400', icon: '↗' }
  return (
    <div className={`w-10 h-10 rounded-xl ${cfg.bg} flex items-center justify-center ${cfg.fg} font-bold text-lg`}>
      {cfg.icon}
    </div>
  )
}

const quickActions = [
  { id: 'deposit', label: 'Depositar', icon: 'M12 6v6m0 0v6m0-6h6m-6 0H6', color: 'from-emerald-500 to-emerald-600', shadow: 'shadow-emerald-500/20' },
  { id: 'withdraw', label: 'Retirar', icon: 'M20 12H4', color: 'from-red-500 to-red-600', shadow: 'shadow-red-500/20' },
  { id: 'transfer', label: 'Transferir', icon: 'M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4', color: 'from-cyan-500 to-cyan-600', shadow: 'shadow-cyan-500/20' },
  { id: 'history', label: 'Historial', icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2', color: 'from-violet-500 to-violet-600', shadow: 'shadow-violet-500/20' },
]

export default function Dashboard() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [account, setAccount] = useState(null)
  const [balance, setBalance] = useState(null)
  const [txs, setTxs] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const fetchAll = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [accRes, balRes, txRes] = await Promise.all([
        client.get('/account'),
        client.get('/account/balance'),
        client.get('/transactions/history?limit=5'),
      ])
      setAccount(accRes.data)
      setBalance(balRes.data.balance)
      setTxs(txRes.data.transactions || [])
    } catch (err) {
      setError(err.response?.data?.error || 'Error al cargar datos')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { fetchAll() }, [fetchAll])

  if (loading && !account) {
    return (
      <div className="max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
        <div className="animate-fade-in">
          <div className="h-8 w-48 bg-white/5 rounded-lg mb-2 animate-pulse" />
          <div className="h-5 w-64 bg-white/5 rounded-lg animate-pulse" />
        </div>
        <div className="glass-card rounded-2xl p-8">
          <div className="h-4 w-24 bg-white/5 rounded mb-4 animate-pulse" />
          <div className="h-12 w-48 bg-white/5 rounded animate-pulse" />
          <div className="grid grid-cols-2 gap-4 mt-6">
            <div className="h-12 bg-white/5 rounded animate-pulse" />
            <div className="h-12 bg-white/5 rounded animate-pulse" />
          </div>
        </div>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {[1,2,3,4].map(i => <div key={i} className="h-24 glass-card rounded-xl animate-pulse" />)}
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-6xl mx-auto px-4 sm:px-6 py-6 sm:py-8 space-y-6 animate-fade-in">
      {error && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-300 text-sm rounded-xl px-4 py-3 flex items-center gap-2">
          <svg className="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          {error}
          <button onClick={fetchAll} className="ml-auto text-xs text-teal-400 hover:text-teal-300 font-medium">Reintentar</button>
        </div>
      )}

      {/* Welcome header */}
      <div className="animate-fade-in-up">
        <div className="flex items-center gap-4 mb-1">
          <div className="w-12 h-12 rounded-2xl bg-gradient-to-br from-teal-400 to-cyan-600 flex items-center justify-center text-white text-lg font-bold shadow-xl shadow-teal-500/20">
            {user?.full_name?.charAt(0)?.toUpperCase() || '?'}
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">Bienvenido, {user?.full_name?.split(' ')[0]}</h1>
            <p className="text-sm text-slate-400 capitalize">{formatDate()}</p>
          </div>
        </div>
      </div>

      {/* Balance card */}
      <div className="animate-fade-in-up" style={{ animationDelay: '0.05s' }}>
        <div className="relative overflow-hidden rounded-2xl bg-gradient-to-br from-slate-800 via-slate-900 to-slate-800 border border-white/10 shadow-2xl">
          <div className="absolute top-0 right-0 w-64 h-64 bg-teal-500/5 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2" />
          <div className="absolute bottom-0 left-0 w-48 h-48 bg-cyan-500/5 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2" />
          <div className="relative p-6 sm:p-8">
            <div className="flex items-start justify-between mb-6">
              <div>
                <p className="text-sm font-medium text-slate-400 uppercase tracking-wider">Saldo disponible</p>
                <p className="text-4xl sm:text-5xl font-bold text-white mt-2 tracking-tight">
                  {balance !== null ? formatCurrency(balance) : '---'}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse-soft" />
                <span className="text-xs text-slate-500">Activa</span>
              </div>
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-4 sm:gap-6">
              <div className="glass-light rounded-xl p-3 sm:p-4">
                <p className="text-xs text-slate-500 mb-0.5">Titular</p>
                <p className="text-sm font-semibold text-white truncate">{user?.full_name}</p>
              </div>
              <div className="glass-light rounded-xl p-3 sm:p-4">
                <p className="text-xs text-slate-500 mb-0.5">Cuenta</p>
                <p className="text-sm font-semibold text-white font-mono">{account?.account_number || user?.account_number}</p>
              </div>
              <div className="glass-light rounded-xl p-3 sm:p-4 col-span-2 sm:col-span-1">
                <p className="text-xs text-slate-500 mb-0.5">Miembro desde</p>
                <p className="text-sm font-semibold text-white">
                  {account?.created_at ? new Date(account.created_at).toLocaleDateString('es-ES', { year: 'numeric', month: 'long' }) : '---'}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Quick actions */}
      <div className="animate-fade-in-up" style={{ animationDelay: '0.1s' }}>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {quickActions.map((action) => (
            <button
              key={action.id}
              onClick={() => navigate(action.id === 'history' ? '/history' : `/transactions?tab=${action.id}`)}
              className="group glass-card rounded-xl p-4 sm:p-5 hover:bg-white/[0.06] transition-all duration-200 active:scale-[0.98]"
            >
              <div className={`w-10 h-10 rounded-xl bg-gradient-to-br ${action.color} ${action.shadow} flex items-center justify-center mb-3 transition-transform duration-200 group-hover:scale-110`}>
                <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d={action.icon} />
                </svg>
              </div>
              <p className="text-sm font-semibold text-slate-200 group-hover:text-white transition-colors">{action.label}</p>
              <p className="text-xs text-slate-500 mt-0.5">Nueva operación</p>
            </button>
          ))}
        </div>
      </div>

      {/* Recent transactions */}
      <div className="animate-fade-in-up" style={{ animationDelay: '0.15s' }}>
        <div className="glass-card rounded-2xl overflow-hidden">
          <div className="p-5 sm:p-6 border-b border-white/5">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-white">Últimos movimientos</h2>
              {txs.length > 0 && (
                <button onClick={() => navigate('/history')} className="text-sm text-teal-400 hover:text-teal-300 font-medium transition-colors">
                  Ver todos
                </button>
              )}
            </div>
          </div>
          {txs.length === 0 ? (
            <div className="text-center py-12 px-6">
              <div className="w-16 h-16 rounded-2xl bg-slate-800/50 flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <p className="text-slate-400 font-medium mb-1">Sin movimientos</p>
              <p className="text-slate-500 text-sm">Realiza un depósito para ver tu actividad aquí.</p>
            </div>
          ) : (
            <div className="divide-y divide-white/5">
              {txs.map((tx) => {
                const isInflow = tx.type === 'deposit'
                const isOutflow = tx.type === 'withdrawal'
                return (
                  <div key={tx.id} className="flex items-center gap-4 p-4 sm:px-6 hover:bg-white/[0.02] transition-colors">
                    <TxIcon type={tx.type} />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-slate-200 capitalize">{tx.type === 'deposit' ? 'Depósito' : tx.type === 'withdrawal' ? 'Retiro' : 'Transferencia'}</p>
                      <p className="text-xs text-slate-500">{formatTime(tx.timestamp)}</p>
                    </div>
                    <p className={`text-sm font-semibold ${isInflow ? 'text-emerald-400' : isOutflow ? 'text-red-400' : 'text-cyan-400'}`}>
                      {isInflow ? '+' : isOutflow ? '−' : '±'}
                      {formatCurrency(tx.amount)}
                    </p>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>

      <ChatWidget />
    </div>
  )
}
