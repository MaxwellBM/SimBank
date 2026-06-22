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
    <div className="max-w-3xl mx-auto px-4 py-8">
      <h1 className="text-2xl font-bold mb-6">Historial de transacciones</h1>

      {error && (
        <div className="bg-red-900/50 border border-red-700 text-red-200 text-sm rounded-lg px-4 py-3 mb-4">{error}</div>
      )}

      {loading ? (
        <div className="space-y-3 animate-pulse">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="h-16 bg-gray-800 rounded-lg" />
          ))}
        </div>
      ) : txs.length === 0 ? (
        <div className="text-center py-16 bg-gray-900 border border-gray-800 rounded-xl">
          <p className="text-gray-400 text-lg mb-1">Sin movimientos</p>
          <p className="text-gray-600 text-sm">Aún no hay transacciones registradas en tu cuenta.</p>
        </div>
      ) : (
        <>
          <div className="space-y-2">
            {displayed.map((tx) => {
              const isInflow = tx.type === 'deposit'
              const isOutflow = tx.type === 'withdrawal'
              return (
                <div key={tx.id} className="bg-gray-900 border border-gray-800 rounded-lg p-4 flex items-center justify-between hover:border-gray-700 transition-colors">
                  <div className="flex items-center gap-4">
                    <div className={`w-10 h-10 rounded-full flex items-center justify-center text-lg ${
                      isInflow ? 'bg-green-900/50 text-green-400' : isOutflow ? 'bg-red-900/50 text-red-400' : 'bg-cyan-900/50 text-cyan-400'
                    }`}>
                      {isInflow ? '+' : isOutflow ? '−' : '↗'}
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-200 capitalize">{tx.type}</p>
                      <p className="text-xs text-gray-500">{formatTime(tx.timestamp)}</p>
                      {tx.type === 'transfer' && (
                        <p className="text-xs text-gray-500">
                          {tx.from_account === '1' ? 'Banco → ' : ''}
                          Cuenta: {tx.from_account === '1' ? tx.to_account.slice(0, 8) + '…' : tx.to_account.slice(0, 8) + '…'}
                        </p>
                      )}
                    </div>
                  </div>
                  <p className={`text-sm font-semibold ${
                    isInflow ? 'text-green-400' : isOutflow ? 'text-red-400' : 'text-cyan-400'
                  }`}>
                    {isInflow ? '+' : isOutflow ? '−' : '±'}
                    {formatCurrency(tx.amount)}
                  </p>
                </div>
              )
            })}
          </div>

          <div className="flex justify-center mt-6 gap-3">
            <button
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              disabled={page === 0}
              className="px-4 py-2 bg-gray-800 hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed text-sm rounded-lg transition-colors"
            >
              ← Anterior
            </button>
            <button
              onClick={() => setPage((p) => p + 1)}
              disabled={!hasMore}
              className="px-4 py-2 bg-gray-800 hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed text-sm rounded-lg transition-colors"
            >
              Siguiente →
            </button>
          </div>
        </>
      )}
    </div>
  )
}
