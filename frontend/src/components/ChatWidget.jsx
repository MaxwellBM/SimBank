import { useState, useRef, useEffect } from 'react'
import client from '../api/client'

export default function ChatWidget() {
  const [messages, setMessages] = useState([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [pendingAction, setPendingAction] = useState(null)
  const [open, setOpen] = useState(false)
  const [configured, setConfigured] = useState(true)
  const bottomRef = useRef(null)

  useEffect(() => { bottomRef.current?.scrollIntoView({ behavior: 'smooth' }) }, [messages])

  const addMessage = (role, content) => {
    setMessages((prev) => [...prev, { role, content }])
  }

  const sendMessage = async () => {
    const text = input.trim()
    if (!text || loading) return
    setInput('')
    addMessage('user', text)
    setLoading(true)

    try {
      const history = messages.map((m) => ({ role: m.role, content: m.content }))
      const { data } = await client.post('/chat', { message: text, history })

      if (data.error) {
        if (data.error.includes('OPENROUTER_API_KEY')) {
          setConfigured(false)
          addMessage('assistant', 'El chat con IA no está configurado. Agrega OPENROUTER_API_KEY en el backend para habilitarlo.')
        } else {
          addMessage('assistant', `Error: ${data.error}`)
        }
        return
      }

      addMessage('assistant', data.reply)

      if (data.requires_confirmation && data.action) {
        setPendingAction(data.action)
      }
    } catch (err) {
      const msg = err.response?.data?.error || 'Error de conexión con el servidor'
      if (msg.includes('OPENROUTER_API_KEY')) {
        setConfigured(false)
        addMessage('assistant', 'El chat con IA no está configurado.')
      } else {
        addMessage('assistant', `Error: ${msg}`)
      }
    } finally {
      setLoading(false)
    }
  }

  const confirmAction = async () => {
    if (!pendingAction) return
    setLoading(true)
    try {
      const { data } = await client.post('/chat/confirm', {
        action_id: pendingAction.id,
        action_type: pendingAction.type,
        to_account: pendingAction.to_account,
        amount: pendingAction.amount,
        description: pendingAction.description || '',
      })
      addMessage('assistant', data.reply || 'Transferencia ejecutada correctamente.')
      setPendingAction(null)
    } catch (err) {
      const msg = err.response?.data?.error || 'Error al confirmar la acción'
      addMessage('assistant', `Error: ${msg}`)
    } finally {
      setLoading(false)
    }
  }

  const cancelAction = () => {
    addMessage('assistant', 'Transferencia cancelada.')
    setPendingAction(null)
  }

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="fixed bottom-4 right-4 bg-cyan-600 hover:bg-cyan-500 text-white rounded-full w-14 h-14 flex items-center justify-center shadow-lg transition-colors z-50"
      >
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
        </svg>
      </button>
    )
  }

  return (
    <div className="fixed bottom-4 right-4 w-96 max-w-[calc(100vw-2rem)] bg-gray-900 border border-gray-700 rounded-xl shadow-2xl flex flex-col z-50" style={{ height: '500px' }}>
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-700">
        <span className="font-semibold text-sm">Asistente IA</span>
        <button onClick={() => setOpen(false)} className="text-gray-400 hover:text-white text-lg leading-none">&times;</button>
      </div>

      <div className="flex-1 overflow-y-auto p-3 space-y-3">
        {messages.length === 0 && (
          <p className="text-gray-500 text-sm text-center mt-8">
            {configured ? 'Pregúntame sobre tu cuenta, saldo, o solicita una transferencia.' : 'Chat no disponible.'}
          </p>
        )}
        {messages.map((m, i) => (
          <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
            <div className={`max-w-[80%] rounded-xl px-3 py-2 text-sm ${
              m.role === 'user' ? 'bg-cyan-700 text-white' : 'bg-gray-800 text-gray-200'
            }`}>
              {m.content}
            </div>
          </div>
        ))}

        {pendingAction && (
          <div className="bg-yellow-900/40 border border-yellow-700 rounded-xl p-3 text-sm">
            <p className="text-yellow-200 font-medium mb-2">⚠ Acción pendiente de confirmación</p>
            <p className="text-gray-300 mb-1">
              <span className="text-gray-400">Tipo:</span> {pendingAction.type}
            </p>
            {pendingAction.to_account && (
              <p className="text-gray-300 mb-1">
                <span className="text-gray-400">Destino:</span> {pendingAction.to_account}
              </p>
            )}
            <p className="text-gray-300 mb-3">
              <span className="text-gray-400">Monto:</span> ${pendingAction.amount.toFixed(2)}
            </p>
            <div className="flex gap-2">
              <button onClick={confirmAction} disabled={loading} className="flex-1 bg-green-700 hover:bg-green-600 disabled:opacity-50 text-white rounded-lg py-1.5 text-sm transition-colors">
                {loading ? 'Procesando...' : 'Confirmar'}
              </button>
              <button onClick={cancelAction} disabled={loading} className="flex-1 bg-red-800 hover:bg-red-700 disabled:opacity-50 text-white rounded-lg py-1.5 text-sm transition-colors">
                Cancelar
              </button>
            </div>
          </div>
        )}

        {loading && !pendingAction && (
          <div className="flex justify-start">
            <div className="bg-gray-800 rounded-xl px-3 py-2 text-sm text-gray-400 flex items-center gap-1">
              <span className="animate-pulse">Escribiendo</span>
              <span className="animate-bounce">.</span>
              <span className="animate-bounce delay-100">.</span>
              <span className="animate-bounce delay-200">.</span>
            </div>
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      <div className="border-t border-gray-700 p-3">
        <div className="flex gap-2">
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && sendMessage()}
            placeholder="Escribe un mensaje..."
            disabled={loading}
            className="flex-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-cyan-600 disabled:opacity-50"
          />
          <button
            onClick={sendMessage}
            disabled={loading || !input.trim()}
            className="bg-cyan-700 hover:bg-cyan-600 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-lg px-3 py-2 transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  )
}
