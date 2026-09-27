// Command dualprobe menguji DualStreamSTT dari ujung ke ujung terhadap
// AssemblyAI sungguhan, tanpa mikrofon dan tanpa UI.
//
// Dipakai ketika gejala di aplikasi tidak cukup untuk menunjuk lapisan mana
// yang gagal: koneksi diarizer, penjodohan waktu, atau pemetaan role.
//
//	go run ./cmd/dualprobe -audio file.pcm
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/bisik/bisik_backend/internal/infrastructure/assemblyai"
)

func main() {
	audioPath := flag.String("audio", "", "berkas PCM16 16 kHz mono")
	textModel := flag.String("text-model", "whisper-rt", "model sumber teks")
	labelModel := flag.String("label-model", "universal-streaming-multilingual", "model sumber label")
	interval := flag.Int("interval", 5000, "speaker_labels_revision_interval_ms")
	flag.Parse()

	key := os.Getenv("ASSEMBLYAI_API_KEY")
	if key == "" || *audioPath == "" {
		fmt.Println("butuh ASSEMBLYAI_API_KEY dan -audio")
		os.Exit(2)
	}
	pcm, err := os.ReadFile(*audioPath)
	if err != nil {
		fmt.Println("baca audio:", err)
		os.Exit(1)
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	const wsURL = "wss://streaming.assemblyai.com/v3/ws"

	transcriber := assemblyai.NewStreamingSTT(key, wsURL, *textModel, *interval, log)
	diarizer := assemblyai.NewStreamingSTT(key, wsURL, *labelModel, *interval, log)
	dual := assemblyai.NewDualStreamSTT(transcriber, diarizer, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const sessionID = "probe-dual"
	events, err := dual.Start(ctx, sessionID)
	if err != nil {
		fmt.Println("start gagal:", err)
		os.Exit(1)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range events {
			kind := "final"
			switch {
			case ev.IsRevision:
				kind = "REVISI"
			case !ev.IsFinal:
				if ev.Acknowledge != nil {
					ev.Acknowledge()
				}
				continue // partial tidak menarik untuk uji ini
			}
			fmt.Printf("%-6s label=%-4q role=%-9s %5d-%-5d %.55s\n",
				kind, ev.SourceSpeaker, ev.Speaker, ev.StartMS, ev.EndMS, ev.Text)
			// Meniru adapter WebSocket: penggabung memblokir sampai diakui.
			if ev.Acknowledge != nil {
				ev.Acknowledge()
			}
		}
	}()

	const chunk = 4096
	started := time.Now()
	pushed := 0
	for i := 0; i < len(pcm); i += chunk {
		end := min(i+chunk, len(pcm))
		if err := dual.PushAudio(sessionID, pcm[i:end]); err != nil {
			fmt.Printf("push gagal pada chunk %d (%.1f detik audio): %v\n",
				pushed, float64(pushed*chunk)/2/16000, err)
			break
		}
		pushed++
		time.Sleep(128 * time.Millisecond)
	}
	fmt.Printf("\n[dorong selesai: %d chunk, %.1f detik audio, %.1f detik nyata]\n",
		pushed, float64(pushed*chunk)/2/16000, time.Since(started).Seconds())

	if err := dual.Stop(sessionID); err != nil {
		fmt.Println("stop:", err)
	}
	select {
	case <-done:
	case <-time.After(25 * time.Second):
		fmt.Println("[timeout]")
	}
}
