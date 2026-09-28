import { useState } from 'react'
import { useI18n } from '../i18n/useI18n'
import { ConfirmDialog } from './ConfirmDialog'

/**
 * Siapa yang sedang masuk, dan jalan keluarnya — dipakai halaman petugas
 * maupun supervisor supaya keduanya berperilaku sama.
 *
 * Keluar selalu dikonfirmasi dulu: tombolnya berdampingan dengan tombol
 * utama di layar yang dipakai sambil berdiri di depan nasabah, dan salah
 * ketuk berarti harus mengetik ulang kata sandi di depan mereka.
 */
export function AccountBar({ name, onLogout }: { name: string; onLogout: () => void }) {
  const { t } = useI18n()
  const [confirming, setConfirming] = useState(false)

  return (
    <>
      <p className="whoami">
        <span>
          {t.auth.signedInAs} <strong>{name}</strong>
        </span>
        <button type="button" className="btn--ghost" onClick={() => setConfirming(true)}>
          {t.auth.logout}
        </button>
      </p>
      <ConfirmDialog
        open={confirming}
        title={t.auth.logoutTitle}
        body={t.auth.logoutBody}
        confirmLabel={t.auth.logout}
        cancelLabel={t.auth.cancel}
        onCancel={() => setConfirming(false)}
        onConfirm={() => {
          setConfirming(false)
          onLogout()
        }}
      />
    </>
  )
}
