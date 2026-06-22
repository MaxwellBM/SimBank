import { useState, useEffect, useCallback } from 'react'
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

export default function Dashboard() {
  const { user } = useAuth()
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

  const txIcon = (type) => {
    switch (type) {
      case 'deposit': return <span className="text-green-400 font-bold">+</span>
      case 'withdrawal': return <span className="text-red-400 font-bold">−</span>
      default: return <span className="text-cyan-400 font-bold">↗</span>
    }
  }

  if (loading) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-8 space-y-6 animate-pulse">
        <div className="h-32 bg-gray-800 rounded-xl" />
        <div className="h-48 bg-gray-800 rounded-xl" />
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto px-4 py-8 space-y-6">
      {error && (
        <div className="bg-red-900/50 border border-red-700 text-red-200 text-sm rounded-lg px-4 py-3">{error}</div>
      )}

      {/* Balance card */}
      <div className="bg-gradient-to-br from-gray-900 to-gray-800 border border-gray-700 rounded-xl p-6">
        <p className="text-gray-400 text-sm">Saldo disponible</p>
        <p className="text-4xl font-bold text-white mt-1">
          {balance !== null ? formatCurrency(balance) : '---'}
        </p>
        <div className="mt-4 grid grid-cols-2 gap-4 text-sm">
          <div>
            <p className="text-gray-500">Titular</p>
            <p className="text-white font-medium">{user?.full_name}</p>
          </div>
          <div>
            <p className="text-gray-500">Cuenta</p>
            <p className="text-white font-medium">{account?.account_number || user?.account_number}</p>
          </div>
        </div>
        <button
          onClick={fetchAll}
          className="mt-4 text-xs text-cyan-400 hover:text-cyan-300 transition-colors"
        >
          ↻ Actualizar
        </button>
      </div>

      {/* Recent transactions */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Últimas transacciones</h2>
        {txs.length === 0 ? (
          <div className="text-center py-10">
            <p className="text-gray-500 text-lg mb-1">No hay transacciones</p>
            <p className="text-gray-600 text-sm">Realiza un depósito o transferencia para ver tu actividad aquí.</p>
          </div>
        ) : (
          <div className="space-y-2">
            {txs.map((tx) => (
              <div key={tx.id} className="flex items-center justify-between py-2 border-b border-gray-800 last:border-0">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-full bg-gray-800 flex items-center justify-center text-sm">
                    {txIcon(tx.type)}
                  </div>
                  <div>
                    <p className="text-sm font-medium text-gray-200 capitalize">{tx.type}</p>
                    <p className="text-xs text-gray-500">{formatTime(tx.timestamp)}</p>
                  </div>
                </div>
                <p className={`text-sm font-medium ${tx.type === 'deposit' ? 'text-green-400' : tx.type === 'withdrawal' ? 'text-red-400' : 'text-cyan-400'}`}>
                  {tx.type === 'deposit' ? '+' : tx.type === 'withdrawal' ? '−' : '±'}{formatCurrency(tx.amount)}
                </p>
              </div>
            ))}
          </div>
        )}
      </div>

      <ChatWidget />
    </div>
  )
}
