import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import client from '../api/client'

const tabs = [
  { id: 'deposit', label: 'Depósito', icon: 'M12 6v6m0 0v6m0-6h6m-6 0H6' },
  { id: 'withdraw', label: 'Retiro', icon: 'M20 12H4' },
  { id: 'transfer', label: 'Transferencia', icon: 'M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4' },
]

export default function Transactions() {
  const [searchParams] = useSearchParams()
  const [tab, setTab] = useState(tabs[0].id)
  const [amount, setAmount] = useState('')
  const [toAccount, setToAccount] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const t = searchParams.get('tab')
    if (t && tabs.some(x => x.id === t)) setTab(t)
  }, [searchParams])

  const reset = () => { setAmount(''); setToAccount(''); setError(''); setSuccess('') }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setSuccess('')

    const amt = parseFloat(amount)
    if (!amount || isNaN(amt) || amt <= 0) { setError('El monto debe ser un número positivo'); return }
    if (tab === 'transfer' && !toAccount.trim()) { setError('La cuenta destino es requerida'); return }

    setLoading(true)
    try {
      const body = { amount: amt }
      if (tab === 'transfer') body.to_account = toAccount.trim()

      await client.post(`/transactions/${tab}`, body)
      const balRes = await client.get('/account/balance')

      setSuccess(`Operación exitosa. Nuevo saldo: ${formatCurrency(balRes.data.balance)}`)
      setAmount('')
      setToAccount('')
    } catch (err) {
      setError(err.response?.data?.error || 'Error al procesar la transacción')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-lg mx-auto px-4 py-6 sm:py-8 animate-fade-in">
      <h1 className="text-2xl font-bold text-white mb-2">Nueva operación</h1>
      <p className="text-slate-400 text-sm mb-6">Selecciona el tipo de transacción que deseas realizar</p>

      {/* Tabs */}
      <div className="glass-card rounded-xl p-1.5 mb-6 flex">
        {tabs.map((t) => (
          <button
            key={t.id}
            onClick={() => { setTab(t.id); reset() }}
            className={`flex-1 flex items-center justify-center gap-2 py-2.5 text-sm font-medium rounded-lg transition-all duration-200 ${
              tab === t.id
                ? 'bg-gradient-to-r from-teal-500 to-cyan-600 text-white shadow-lg shadow-teal-500/20'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d={t.icon} />
            </svg>
            {t.label}
          </button>
        ))}
      </div>

      {/* Form */}
      <form onSubmit={handleSubmit} className="glass-card rounded-2xl p-6 sm:p-8 space-y-5">
        {error && (
          <div className="bg-red-500/10 border border-red-500/20 text-red-300 text-sm rounded-xl px-4 py-3 flex items-center gap-2">
            <svg className="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            {error}
          </div>
        )}
        {success && (
          <div className="bg-emerald-500/10 border border-emerald-500/20 text-emerald-300 text-sm rounded-xl px-4 py-3 flex items-center gap-2">
            <svg className="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            {success}
          </div>
        )}

        {tab === 'transfer' && (
          <div className="space-y-1.5">
            <label className="block text-sm font-medium text-slate-300">Cuenta destino</label>
            <input
              type="text"
              value={toAccount}
              onChange={(e) => setToAccount(e.target.value)}
              className="w-full bg-slate-800/50 border border-white/10 rounded-xl px-4 py-2.5 text-white placeholder-slate-500 focus:outline-none focus:border-teal-500/50 focus:ring-1 focus:ring-teal-500/20 transition-all duration-200 font-mono"
              placeholder="001-0002"
            />
          </div>
        )}

        <div className="space-y-1.5">
          <label className="block text-sm font-medium text-slate-300">
            Monto {tab === 'transfer' ? 'a transferir' : tab === 'deposit' ? 'a depositar' : 'a retirar'}
          </label>
          <div className="relative">
            <span className="absolute left-4 top-1/2 -translate-y-1/2 text-slate-400 font-semibold text-lg">$</span>
            <input
              type="number"
              step="0.01"
              min="0.01"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              className="w-full bg-slate-800/50 border border-white/10 rounded-xl pl-8 pr-4 py-2.5 text-white placeholder-slate-500 focus:outline-none focus:border-teal-500/50 focus:ring-1 focus:ring-teal-500/20 transition-all duration-200 text-lg"
              placeholder="0.00"
              autoFocus
            />
          </div>
        </div>

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-gradient-to-r from-teal-500 to-cyan-600 hover:from-teal-400 hover:to-cyan-500 disabled:opacity-50 disabled:cursor-not-allowed text-white font-semibold rounded-xl py-3 transition-all duration-200 shadow-lg shadow-teal-500/20 hover:shadow-teal-500/30 active:scale-[0.98]"
        >
          {loading ? (
            <span className="flex items-center justify-center gap-2">
              <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              Procesando...
            </span>
          ) : `${tabs.find(t => t.id === tab)?.label || 'Enviar'}`
          }
        </button>
      </form>
    </div>
  )
}

function formatCurrency(n) {
  return '$' + Number(n).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}
