import { useState, useEffect, useCallback } from 'react'
import client from '../api/client'

const PAGE_SIZE = 10

function formatCurrency(n) {
  return '$' + Number(n).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function formatTime(ts) {
  return new Date(ts).toLocaleDateString('es-ES', {
    day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit',
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

export default function History() {
  const [txs, setTxs] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [page, setPage] = useState(0)

  const fetchHistory = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const limit = (page + 1) * PAGE_SIZE
      const { data } = await client.get(`/transactions/history?limit=${limit}`)
      setTxs(data.transactions || [])
    } catch (err) {
      setError(err.response?.data?.error || 'Error al cargar historial')
    } finally {
      setLoading(false)
    }
  }, [page])

  useEffect(() => { fetchHistory() }, [fetchHistory])

  const displayed = txs.slice(0, (page + 1) * PAGE_SIZE)
  const hasMore = txs.length > (page + 1) * PAGE_SIZE

  return (
    <div className="max-w-3xl mx-auto px-4 py-6 sm:py-8 animate-fade-in">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-white">Historial</h1>
        <p className="text-slate-400 text-sm mt-1">Todas las transacciones de tu cuenta</p>
      </div>

      {error && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-300 text-sm rounded-xl px-4 py-3 flex items-center gap-2 mb-4">
          <svg className="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          {error}
        </div>
      )}

      {loading && txs.length === 0 ? (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="glass-card rounded-xl p-4 animate-pulse">
              <div className="flex items-center gap-4">
                <div className="w-10 h-10 rounded-xl bg-white/5" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 w-24 bg-white/5 rounded" />
                  <div className="h-3 w-32 bg-white/5 rounded" />
                </div>
                <div className="h-4 w-20 bg-white/5 rounded" />
              </div>
            </div>
          ))}
        </div>
      ) : txs.length === 0 ? (
        <div className="glass-card rounded-2xl text-center py-16 px-6">
          <div className="w-16 h-16 rounded-2xl bg-slate-800/50 flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
          </div>
          <p className="text-slate-400 font-medium mb-1">Sin movimientos</p>
          <p className="text-slate-500 text-sm">Aún no hay transacciones registradas en tu cuenta.</p>
        </div>
      ) : (
        <>
          <div className="space-y-2">
            {displayed.map((tx, i) => {
              const isInflow = tx.type === 'deposit'
              const isOutflow = tx.type === 'withdrawal'
              return (
                <div
                  key={tx.id}
                  className="glass-card rounded-xl p-4 flex items-center gap-4 hover:bg-white/[0.03] transition-all duration-200 animate-fade-in"
                  style={{ animationDelay: `${i * 0.03}s` }}
                >
                  <TxIcon type={tx.type} />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-slate-200 capitalize">
                      {tx.type === 'deposit' ? 'Depósito' : tx.type === 'withdrawal' ? 'Retiro' : 'Transferencia'}
                    </p>
                    <p className="text-xs text-slate-500">{formatTime(tx.timestamp)}</p>
                    {tx.type === 'transfer' && tx.description && (
                      <p className="text-xs text-slate-500 truncate mt-0.5">{tx.description}</p>
                    )}
                  </div>
                  <p className={`text-sm font-semibold whitespace-nowrap ${
                    isInflow ? 'text-emerald-400' : isOutflow ? 'text-red-400' : 'text-cyan-400'
                  }`}>
                    {isInflow ? '+' : isOutflow ? '−' : '±'}
                    {formatCurrency(tx.amount)}
                  </p>
                </div>
              )
            })}
          </div>

          <div className="flex items-center justify-between mt-6">
            <p className="text-sm text-slate-500">
              Mostrando {displayed.length} de {txs.length} transacciones
            </p>
            <div className="flex gap-2">
              <button
                onClick={() => setPage(p => Math.max(0, p - 1))}
                disabled={page === 0}
                className="px-4 py-2 glass-light rounded-xl text-sm font-medium text-slate-300 hover:text-white hover:bg-white/10 disabled:opacity-30 disabled:cursor-not-allowed transition-all duration-200"
              >
                ← Anterior
              </button>
              <button
                onClick={() => setPage(p => p + 1)}
                disabled={!hasMore}
                className="px-4 py-2 glass-light rounded-xl text-sm font-medium text-slate-300 hover:text-white hover:bg-white/10 disabled:opacity-30 disabled:cursor-not-allowed transition-all duration-200"
              >
                Siguiente →
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
