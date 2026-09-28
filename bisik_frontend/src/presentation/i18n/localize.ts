import type { Role } from '../../domain/entities/auth'
import type { Nudge, Obligation } from '../../domain/entities/compliance'
import { AUDIO_REASONS } from '../../domain/repositories/audio_repository'
import type { Dictionary } from './dictionary'

/** Label kewajiban menurut kodenya; kode baru dari backend memakai label aslinya. */
export function obligationLabel(t: Dictionary, o: Pick<Obligation, 'code' | 'label'>): string {
  return t.obligations.label[o.code] ?? o.label
}

/**
 * Menerjemahkan kode `err:…` buatan klien (lihat `AppError`).
 * Pesan lain — biasanya dari server — dikembalikan apa adanya.
 */
export function localizeError(t: Dictionary, raw: string): string {
  if (!raw.startsWith('err:')) return raw
  const [, code, ...args] = raw.split(':')
  switch (code) {
    case 'invalid-credentials':
      return t.errors.invalidCredentials
    case 'login-failed':
      return t.errors.loginFailed(args[0] ?? '?')
    case 'role-mismatch':
      return t.errors.roleMismatch(t.roles[args[0] as Role] ?? args[0], t.roles[args[1] as Role] ?? args[1])
    case 'gateway-lost':
      return t.errors.gatewayLost
    default:
      return raw
  }
}

const audioReasonKey = new Map(
  (Object.keys(AUDIO_REASONS) as Array<keyof typeof AUDIO_REASONS>).map((key) => [
    AUDIO_REASONS[key] as string,
    key,
  ]),
)

/** Alasan audio dari gateway selalu berbahasa Indonesia; terjemahkan yang dikenali. */
export function localizeAudioReason(t: Dictionary, reason: string): string {
  const key = audioReasonKey.get(reason)
  return key ? t.audio[key] : reason
}

/** Teks bisikan yang tampil di layar. Yang diucapkan tetap `nudge.text`. */
export function nudgeText(t: Dictionary, nudge: Nudge): string {
  switch (nudge.kind) {
    case 'avoid_phrase':
      return nudge.phrase ? t.nudge.avoidPhrase(nudge.phrase) : nudge.text
    case 'pending_obligation':
      return nudge.code
        ? t.nudge.pending(obligationLabel(t, { code: nudge.code, label: nudge.text }))
        : nudge.text
    case 'demo_correction':
      return t.demo.correction
    case 'demo_reminder':
      return t.demo.reminder
    default:
      return nudge.text
  }
}
