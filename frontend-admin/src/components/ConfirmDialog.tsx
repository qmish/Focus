import { useEffect, useRef } from 'react'

type Props = {
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  onConfirm: () => void
  onCancel: () => void
  inputMode?: boolean
  inputValue?: string
  onInputChange?: (val: string) => void
}

export default function ConfirmDialog({ open, title, message, confirmLabel = 'OK', cancelLabel = 'Отмена', onConfirm, onCancel, inputMode, inputValue, onInputChange }: Props) {
  const dialogRef = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    if (open) dialogRef.current?.showModal()
    else dialogRef.current?.close()
  }, [open])

  if (!open) return null

  return (
    <dialog ref={dialogRef} onClose={onCancel} role="dialog" aria-modal="true" style={{ border: 'none', borderRadius: 12, padding: '1.5rem', maxWidth: 400, boxShadow: '0 8px 32px rgba(0,0,0,0.2)' }}>
      <h3 style={{ margin: '0 0 0.75rem' }}>{title}</h3>
      <p style={{ margin: '0 0 1rem', color: '#555' }}>{message}</p>
      {inputMode && (
        <input
          type="text"
          value={inputValue || ''}
          onChange={e => onInputChange?.(e.target.value)}
          style={{ width: '100%', padding: '0.5rem', marginBottom: '1rem', border: '1px solid #ccc', borderRadius: 6, boxSizing: 'border-box' }}
          autoFocus
        />
      )}
      <div style={{ display: 'flex', gap: '0.5rem', justifyContent: 'flex-end' }}>
        <button onClick={onCancel} style={{ padding: '0.5rem 1rem', border: '1px solid #ccc', borderRadius: 6, background: '#fff', cursor: 'pointer' }}>{cancelLabel}</button>
        <button onClick={onConfirm} style={{ padding: '0.5rem 1rem', border: 'none', borderRadius: 6, background: '#2563eb', color: '#fff', cursor: 'pointer' }}>{confirmLabel}</button>
      </div>
    </dialog>
  )
}
