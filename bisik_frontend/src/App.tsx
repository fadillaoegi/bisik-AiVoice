import { useEffect } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { OfficerPage } from './presentation/pages/officer/OfficerPage'
import { SupervisorPage } from './presentation/pages/supervisor/SupervisorPage'
import { container } from './infrastructure/di/container'
import { useAppDispatch } from './presentation/hooks/redux'
import { authRestored } from './application/store/slices/authSlice'

export default function App() {
  const dispatch = useAppDispatch()

  // Token tersimpan di perangkat, tetapi hanya server yang bisa memutuskan
  // masih sah atau tidak. Selama pemeriksaan berjalan, halaman menampilkan
  // status memuat alih-alih berkedip ke layar login lebih dulu.
  useEffect(() => {
    container.repositories.auth
      .restore()
      .then((user) => dispatch(authRestored(user)))
      .catch(() => dispatch(authRestored(null)))
  }, [dispatch])

  return (
    <Routes>
      <Route path="/" element={<Navigate to="/officer" replace />} />
      <Route path="/officer" element={<OfficerPage />} />
      <Route path="/supervisor" element={<SupervisorPage />} />
    </Routes>
  )
}
