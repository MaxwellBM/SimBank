import { useState } from 'react'
import client from '../api/client'

const tabs = ['deposit', 'withdraw', 'transfer']

export default function Transactions() {
  const [tab, setTab] = useState('deposit')
  const [amount, setAmount] = useState('')
  const [toAccount, setToAccount] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const reset = () => {
    setAmount('')
    setToAccount('')
    setMessage('')
    setError('')
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setMessage('')

    const amt = parseFloat(amount)
    if (!amount || isNaN(amt) || amt <= 0) {
      setError('El monto debe ser un número positivo')
      return
    }
    if (tab === 'transfer' && !toAccount.trim()) {
      setError('El número de cuenta destino es requerido')
      return
    }

    setLoading(true)
    try {
      let body = { amount: amt }
      if (tab === 'transfer') body.to_account = toAccount.trim()

      const { data } = await client.post(`/transactions/${tab}`, body)

      // Refresh balance
      const balRes = await client.get('/account/balance')

      setMessage(`${tab === 'deposit' ? 'Depósito' : tab === 'withdraw' ? 'Retiro' : 'Transferencia'} exitoso. Nuevo saldo: $${balRes.data.balance.toFixed(2)}`)
      setAmount('')
      setToAccount('')
    } catch (err) {
      setError(err.response?.data?.error || 'Error al procesar la transacción')
    } finally {
      setLoading(false)
    }
  }

  const tabLabel = { deposit: 'Depósito', withdraw: 'Retiro', transfer: 'Transferencia' }

  return (
    <div className="max-w-lg mx-auto px-4 py-8">
      <h1 className="text-2xl font-bold mb-6">Transacciones</h1>

      {/* Tabs */}
      <div className="flex bg-gray-900 rounded-lg p-1 mb-6">
        {tabs.map((t) => (
          <button
            key={t}
            onClick={() => { setTab(t); reset() }}
            className={`flex-1 py-2 text-sm font-medium rounded-md transition-colors ${
              tab === t ? 'bg-cyan-700 text-white' : 'text-gray-400 hover:text-white'
            }`}
          >
            {tabLabel[t]}
          </button>
        ))}
      </div>

      {/* Form */}
      <form onSubmit={handleSubmit} className="bg-gray-900 border border-gray-800 rounded-xl p-6 space-y-4">
        {error && (
          <div className="bg-red-900/50 border border-red-700 text-red-200 text-sm rounded-lg px-3 py-2">{error}</div>
        )}
        {message && (
          <div className="bg-green-900/50 border border-green-700 text-green-200 text-sm rounded-lg px-3 py-2">{message}</div>
        )}

        {tab === 'transfer' && (
          <div>
            <label className="block text-sm text-gray-400 mb-1">Cuenta destino</label>
            <input
              type="text"
              value={toAccount}
              onChange={(e) => setToAccount(e.target.value)}
              className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white focus:outline-none focus:border-cyan-600"
              placeholder="001-0002"
            />
          </div>
        )}

        <div>
          <label className="block text-sm text-gray-400 mb-1">Monto ($)</label>
          <input
            type="number"
            step="0.01"
            min="0.01"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            className="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-white focus:outline-none focus:border-cyan-600"
            placeholder="0.00"
            autoFocus
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-cyan-700 hover:bg-cyan-600 disabled:opacity-50 text-white font-medium rounded-lg py-2 transition-colors"
        >
          {loading ? 'Procesando...' : `${tabLabel[tab]}`}
        </button>
      </form>
    </div>
  )
}
