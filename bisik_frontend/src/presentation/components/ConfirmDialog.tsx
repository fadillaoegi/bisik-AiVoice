import { useEffect, useRef } from 'react'

interface Props {
  open: boolean
  title: string
  body: string
  confirmLabel: string
  cancelLabel: string
  onConfirm: () => void
  onCancel: () => void
}

/**
 * Dialog konfirmasi di atas elemen `<dialog>` bawaan peramban.
 *
 * Dipilih ketimbang div buatan sendiri karena `showModal()` sudah memberi
 * semuanya gratis: latar diblokir, fokus terkunci di dalam dialog, dan Esc
 * menutupnya. Fokus awal jatuh ke "Batal" — pilihan yang aman kalau petugas
 * menekan Enter tanpa sengaja.
 */
export function ConfirmDialog({
  open,
  title,
  body,
  confirmLabel,
  cancelLabel,
  onConfirm,
  onCancel,
}: Props) {
  const ref = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    const dialog = ref.current
    if (!dialog) return
    if (open && !dialog.open) dialog.showModal()
    if (!open && dialog.open) dialog.close()
  }, [open])

  return (
    <dialog
      ref={ref}
      className="confirm"
      aria-labelledby="confirm-title"
      aria-describedby="confirm-body"
      // Esc memicu `cancel`; cegah penutupan bawaan supaya state React
      // tetap satu-satunya sumber kebenaran.
      onCancel={(e) => {
        e.preventDefault()
        onCancel()
      }}
      // Klik di latar (di luar kotak) sama dengan Batal.
      onClick={(e) => {
        if (e.target === e.currentTarget) onCancel()
      }}
    >
      <div className="confirm__box">
        <h2 id="confirm-title" className="confirm__title">{title}</h2>
        <p id="confirm-body" className="confirm__body">{body}</p>
        <div className="confirm__actions">
          <button type="button" className="btn btn--secondary" onClick={onCancel} autoFocus>
            {cancelLabel}
          </button>
          <button type="button" className="btn btn--danger" onClick={onConfirm}>
            {confirmLabel}
          </button>
        </div>
      </div>
    </dialog>
  )
}
