# Bisik

> A private, preventive compliance copilot for face-to-face lending conversations.

Bisik listens to **two humans**—a field loan officer and a customer—identifies
who is speaking, tracks which disclosures the officer has actually made, and
whispers the next missing obligation privately into the officer's earpiece
before the customer signs.

Built for the **AssemblyAI Voice Agent Hackathon 2026**.

## Judge links

| Resource | Link |
|---|---|
| Live application | [https://fldevboo.biz.id](https://fldevboo.biz.id) |
| Officer experience | [https://fldevboo.biz.id/officer](https://fldevboo.biz.id/officer) |
| Supervisor dashboard | [https://fldevboo.biz.id/supervisor](https://fldevboo.biz.id/supervisor) |
| Repository | [github.com/fadillaoegi/bisik-AiVoice](https://github.com/fadillaoegi/bisik-AiVoice) |

The root URL redirects to the officer experience. The web interface supports
Bahasa Indonesia and English.

## Demo credentials

These accounts are intentionally public and exist only for hackathon judging.
Both were verified against the deployed application on 30 September 2026.

| Role | Username | Password | Opens |
|---|---|---|---|
| Officer | `petugas` | `petugas123` | `/officer` |
| Supervisor | `supervisor` | `supervisor123` | `/supervisor` |

The guided demo below does **not** require an account.

## Fastest judging path: 11-second guided demo

1. Open the [officer page](https://fldevboo.biz.id/officer).
2. Switch to **EN** if preferred.
3. Click **Play guided demo · 11 seconds**.
4. Watch the checklist during the intentional speaker-label error.

The guided demo runs entirely in the browser—no login, microphone, backend, or
external API is required. It is visibly labelled **LOCAL SIMULATION** and shows:

1. the officer making a forbidden promise;
2. a private correction sent only to the officer;
3. obligations being completed with transcript evidence;
4. an officer sentence temporarily labelled as customer speech;
5. the checklist correctly refusing to turn green;
6. a later speaker-label revision and evidence re-evaluation; and
7. the final evidence-backed report.

The fourth and fifth steps are the important safe-fail proof: customer speech,
unknown speech, and an uncertain attribution can never satisfy an officer's
obligation.

## Test a live two-person session

For the most reliable live test:

1. Use Chrome and sign in at `/officer` with the officer account above.
2. Wear an earphone so Bisik's private whisper is not captured by the mic.
3. Click **Start Session** and allow microphone access.
4. Let the officer and customer read their calibration sentence one at a time.
5. Confirm the two voice roles before scoring begins.
6. Speak in turns, leaving a one- or two-second gap between speakers.
7. Use Bahasa Indonesia for the officer's disclosure sentences. The current
   evidence matcher is Indonesia-first; the customer may speak either language.
8. End the session only after the last utterance appears in the transcript.

Suggested officer lines:

```text
Selamat pagi, Pak. Perkenalkan, nama saya Rina dari Bank Nusantara.
Tenang saja, Pak, pengajuan ini pasti disetujui.              # intentional violation
Maaf, saya koreksi. Persetujuan tetap melalui penilaian dulu, Pak.
Suku bunganya 1,2 persen setiap bulan, dengan biaya administrasi seratus ribu rupiah.
Tenornya 12 bulan, dengan cicilan sekitar 953 ribu rupiah per bulan.
Kalau terlambat membayar, ada denda lima ribu rupiah per hari.
Bapak berhak menolak atau membatalkan penawaran ini.
```

Expected result: five satisfied obligations, one forbidden-promise violation,
and a final score of **90/100**, with a quote and timestamp for every satisfied
item. The complete two-person script is in
[`submission/skrip-uji-dialog-dwibahasa.txt`](./submission/skrip-uji-dialog-dwibahasa.txt).

## Supervisor path

1. Sign in at `/supervisor` with the supervisor account.
2. Select a recent session.
3. Review speakers, checklist progress, warnings, violations, and score.
4. Open the completed report to inspect evidence quotes and timestamps.

The supervisor never sends microphone audio and never receives the officer's
private whisper.

## The problem

In Indonesian field lending, consequential product explanations often happen
face to face, from memory, without a supervisor in the room. If an officer
forgets the interest rate, repayment term, late penalty, or the customer's
right to walk away, the omission may only be discovered after the agreement is
signed.

Traditional compliance review is forensic: it explains what went wrong later.
Bisik is preventive: it helps the officer correct the conversation while the
customer can still make an informed decision.

## What makes Bisik different

- **Private coaching, not public interruption.** Corrections are spoken through
  the officer's own device and earphone.
- **Preventive, not post-facto.** The next missing duty is surfaced before the
  conversation finishes.
- **Two-human listening.** Bisik is not another human-to-bot assistant; it must
  distinguish evidence from the officer, customer, and unknown speakers.
- **Evidence, not a green checkbox.** Every satisfied item retains the exact
  sentence and timestamp that justified it.
- **Safe when uncertain.** A delayed check is preferable to a false claim that
  the officer said something they never said.
- **Indonesia-first field workflow.** The demo uses a transparent five-item SOP
  for Indonesian lending conversations.

## How AssemblyAI is used

Audio is captured as PCM16 16 kHz mono and sent through the Go gateway to two
AssemblyAI streaming paths:

- `whisper-rt` produces accurate Indonesian real-time transcription;
- streaming multilingual diarization separates and revises speaker labels;
- word spans and `SpeakerRevision` events are merged before compliance scoring;
- explicit two-voice calibration maps anonymous labels to officer/customer;
- calibration speech is never stored or scored.

AssemblyAI provides the real-time speech and speaker intelligence. Bisik adds
the role-confirmation protocol, deterministic guardrails, evidence gates,
semantic verification, private browser TTS, safe-fail behaviour, and reporting.

Semantic verification is provider-independent: Groq is currently first, with
configured fallbacks. It is called only after a deterministic evidence gate and
must return at least `0.80` confidence.

## Runtime flow

```text
Officer microphone (PWA AudioWorklet / Flutter recorder)
       │ PCM16, 16 kHz mono
       ▼
Go WebSocket gateway
       ├── AssemblyAI transcription stream
       ├── AssemblyAI speaker-diarization stream
       └── merge text + speaker spans + later revisions
                         │
                         ▼
              confirmed speaker roles
                         │
        ┌────────────────┴─────────────────┐
        ▼                                  ▼
deterministic guardrail          evidence gate + semantic verifier
        │                                  │
        └──────── private nudge ───────────┘
                         │
                         ▼
 PostgreSQL transcript, evidence, violations, final report
```

## Five demo SOP obligations

| Code | Required disclosure |
|---|---|
| `IDENTITY` | Officer identity and institution |
| `RATE` | Interest rate or total cost |
| `TENOR` | Term and instalment amount |
| `PENALTY` | Late-payment penalty |
| `RIGHT` | Customer's right to refuse or cancel |

These are explicitly a **hackathon demo SOP**, not a claim that the prototype
implements or certifies official Indonesian regulation.

## Safe-fail rules

| Situation | Behaviour |
|---|---|
| Specific evidence is missing | Rejected before semantic verification |
| Semantic confidence is below `0.80` | Obligation remains pending |
| Forbidden promise is spoken | Deterministic guardrail records it immediately |
| Voice roles are not calibrated | Compliance scoring remains locked |
| Customer or unknown speaker says the right words | Checklist does not advance |
| A third voice appears | Kept `unknown`; never inherits another role |
| Audio is quiet, clipped, or noisy | Checklist is held and the officer is warned |
| Speaker label is revised | The affected evidence is evaluated again |
| Verifier or upstream service fails | UI reports the failure; no false-green result |
| A spoken whisper echoes into the mic | Recent-nudge echo guard excludes it from scoring |

## Privacy and access control

- Raw audio is never stored. PostgreSQL keeps transcripts, evidence, session
  state, and violations.
- Customers do not log in, hold the device, or hear the private whisper.
- Officer and supervisor permissions are enforced by signed backend tokens.
- A report is readable only by its officer or a supervisor.
- Session ownership comes from the token, never from a client-supplied owner ID.
- Passwords are stored with PBKDF2-HMAC-SHA256 (600,000 iterations).

Customer consent for transcription would be the deploying institution's
responsibility; a production consent workflow is outside this prototype.

## Repository structure

| Project | Purpose | Stack |
|---|---|---|
| [`bisik_frontend/`](./bisik_frontend) | Primary judge experience: officer PWA, supervisor dashboard, report, guided demo | React 19, TypeScript, Vite, Redux Toolkit, Workbox |
| [`bisik_backend/`](./bisik_backend) | WebSocket gateway, AssemblyAI streams, auth, rules, evidence, reporting | Go, pgx, PostgreSQL 18 |
| [`bisik_mobile/`](./bisik_mobile) | Field officer mobile client | Flutter, Riverpod, Dio |

All three projects follow the same dependency direction:

```text
domain  ←  application/use cases  ←  adapters/infrastructure  ←  presentation
```

See [`bisik_backend/README.md`](./bisik_backend/README.md) for backend boundaries
and endpoints, and [`bisik_backend/SCHEMA.md`](./bisik_backend/SCHEMA.md) for the
database schema and migrations.

## Run locally

Requirements: Go, Node 24+, pnpm, Flutter for the mobile client, and Docker only
for PostgreSQL.

### 1. PostgreSQL and backend

```bash
cd bisik_backend
test -f .env || cp .env.example .env
# Fill ASSEMBLYAI_API_KEY, AUTH_SECRET, and at least one semantic-provider key.
make db
make dev
```

The API starts on `http://localhost:8080`.

### 2. React PWA

```bash
cd bisik_frontend
cp .env.example .env
pnpm install
pnpm dev
```

Open `http://localhost:5173/officer`.

### 3. Flutter client

```bash
cd bisik_mobile
flutter pub get
flutter run \
  --dart-define=API_URL=http://10.0.2.2:8080 \
  --dart-define=WS_URL=ws://10.0.2.2:8080
```

`10.0.2.2` is the host address from an Android emulator. For an iPhone,
physical Android device, or iOS simulator, follow [`TESTING.md`](./TESTING.md)
and use the correct host/LAN address.

## Verification

```bash
cd bisik_backend && go test ./... && go vet ./...
cd ../bisik_frontend && pnpm lint && pnpm build
cd ../bisik_mobile && flutter analyze && flutter test
```

The deployment image serves the compiled PWA and Go API from one container;
PostgreSQL runs separately. Frontend production assets are built with host
Node—there is no Node Docker image in this repository.

## Honest limitations

- The compliance evidence rules are currently Indonesia-first even when the UI
  is switched to English.
- Overlapping speech captured by one shared microphone cannot be separated
  perfectly; the demo asks participants to speak in turns.
- Audio-quality thresholds were selected conservatively and still need broader
  calibration across real rooms and devices.
- The dual AssemblyAI stream has been proven with real two-voice audio, but the
  full two-person browser workflow has not been repeated as a formal benchmark.
- The Flutter client builds for Android and iOS but the React PWA is the primary
  judging experience.
- The public usernames and passwords above are disposable hackathon accounts
  and must be removed or rotated for any real deployment.

## Documentation

- [`TESTING.md`](./TESTING.md) — complete test/runbook and troubleshooting
- [`PROGRESS.md`](./PROGRESS.md) — chronological engineering decisions and QA
- [`FEATURE-ROADMAP.md`](./FEATURE-ROADMAP.md) — feature status and priorities
- [`submission/`](./submission) — pitch copy, video script, dialogue, and deck

---

**Bisik: an AI that listens to two humans, so the right thing is said before
the customer signs.**
