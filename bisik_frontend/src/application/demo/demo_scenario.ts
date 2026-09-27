import type { ComplianceReport, Obligation, Violation } from '../../domain/entities/compliance'
import type { Session, Utterance } from '../../domain/entities/session'

/**
 * Skenario lokal untuk juri saat layanan eksternal atau mikrofon tidak tersedia.
 * Semua pemakaiannya wajib diberi label "simulasi"; data ini bukan keluaran AssemblyAI.
 */
export const demoObligations: Obligation[] = [
  { code: 'IDENTITY', label: 'Identitas & lembaga', status: 'pending', confidence: 0 },
  { code: 'RATE', label: 'Suku bunga / biaya', status: 'pending', confidence: 0 },
  { code: 'TENOR', label: 'Jangka waktu & cicilan', status: 'pending', confidence: 0 },
  { code: 'PENALTY', label: 'Denda keterlambatan', status: 'pending', confidence: 0 },
  { code: 'RIGHT', label: 'Hak membatalkan', status: 'pending', confidence: 0 },
]

export const demoUtterances: Utterance[] = [
  {
    id: 'demo-1',
    speaker: 'officer',
    text: 'Selamat siang, saya Rina dari Bank Nusantara.',
    startMs: 0,
    revised: false,
  },
  {
    id: 'demo-2',
    speaker: 'customer',
    text: 'Kalau saya ajukan hari ini, apakah pasti diterima?',
    startMs: 3_200,
    revised: false,
  },
  {
    id: 'demo-3',
    speaker: 'officer',
    text: 'Tenang, pengajuan Ibu pasti disetujui.',
    startMs: 6_100,
    revised: false,
  },
  {
    id: 'demo-4',
    speaker: 'officer',
    text: 'Maaf, persetujuan tetap mengikuti penilaian. Bunganya 1,2 persen per bulan, dengan tenor 12 bulan dan cicilan sekitar satu juta rupiah per bulan.',
    startMs: 8_800,
    revised: false,
  },
  {
    id: 'demo-5',
    speaker: 'officer',
    text: 'Jika terlambat ada denda 0,1 persen per hari. Ibu juga berhak menolak atau membatalkan penawaran ini.',
    startMs: 14_200,
    revised: false,
  },
]

/**
 * Babak revisi diarization.
 *
 * Kalimat terakhir petugas mula-mula salah dilabeli sebagai nasabah — ini
 * kejadian nyata pada streaming diarization, bukan dramatisasi. Selama label
 * itu masih `customer`, kedua kewajibannya TIDAK boleh terpenuhi: hanya
 * ucapan petugas yang bisa memenuhi checklist.
 *
 * Ketika AssemblyAI mengirim `SpeakerRevision` dan labelnya berubah menjadi
 * petugas, ucapan itu belum pernah dinilai — jadi harus dinilai sekarang.
 * Tanpa penilaian ulang, koreksi diarization justru membuat laporan salah.
 */
export const demoMislabeledUtterance: Utterance = {
  ...demoUtterances[4],
  speaker: 'customer',
}

export const demoViolation: Violation = {
  phrase: 'pasti disetujui',
  severity: 'high',
  evidenceId: 'demo-3',
  detectedAt: '2026-09-24T10:00:06+07:00',
}

const completedObligations: Obligation[] = [
  { ...demoObligations[0], status: 'satisfied', confidence: 0.99, evidenceId: 'demo-1' },
  { ...demoObligations[1], status: 'satisfied', confidence: 0.96, evidenceId: 'demo-4' },
  { ...demoObligations[2], status: 'satisfied', confidence: 0.95, evidenceId: 'demo-4' },
  { ...demoObligations[3], status: 'satisfied', confidence: 0.94, evidenceId: 'demo-5' },
  { ...demoObligations[4], status: 'satisfied', confidence: 0.98, evidenceId: 'demo-5' },
]

export function createDemoSession(status: Session['status'] = 'active'): Session {
  return {
    id: 'demo-simulasi-001',
    officerId: 'PTG-DEMO',
    productId: 'KREDIT-MULTIGUNA',
    status,
    startedAt: new Date().toISOString(),
    score: status === 'ended' ? 90 : 0,
  }
}

export function createDemoReport(session: Session): ComplianceReport {
  return {
    session: { ...session, status: 'ended', score: 90 },
    obligations: completedObligations,
    violations: [demoViolation],
    // Baris terakhir membawa tanda `revised`: laporan menampilkan label
    // yang sudah dikoreksi, bukan tebakan pertama diarization.
    transcript: demoUtterances.map((u) =>
      u.id === 'demo-5' ? { ...u, revised: true } : u,
    ),
  }
}
