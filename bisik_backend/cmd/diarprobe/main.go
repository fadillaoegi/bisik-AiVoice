// Command diarprobe menguji apakah kombinasi speech model + speaker
// diarization benar-benar menghasilkan label pembicara.
//
// Alat diagnosis, bukan bagian aplikasi. Dia mengirim PCM16 16 kHz mono ke
// AssemblyAI persis seperti gateway, lalu MENCETAK SETIAP PESAN APA ADANYA —
// tanpa parsing, tanpa asumsi. Dipakai ketika gejala di aplikasi sudah
// mentok dan yang dibutuhkan adalah fakta dari server.
//
//	go run ./cmd/diarprobe -audio file.pcm -model whisper-rt
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	audioPath := flag.String("audio", "", "berkas PCM16 16 kHz mono")
	model := flag.String("model", "whisper-rt", "speech_model")
	interval := flag.Int("interval", 5000, "speaker_labels_revision_interval_ms")
	maxSpeakers := flag.Int("max-speakers", 3, "max_speakers")
	extra := flag.String("extra", "", "parameter tambahan, mis. language_code=id&format_turns=true")
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

	q := url.Values{}
	q.Set("speech_model", *model)
	q.Set("encoding", "pcm_s16le")
	q.Set("sample_rate", "16000")
	q.Set("speaker_labels", "true")
	q.Set("max_speakers", strconv.Itoa(*maxSpeakers))
	q.Set("speaker_labels_revision_interval_ms", strconv.Itoa(*interval))

	if *extra != "" {
		more, err := url.ParseQuery(*extra)
		if err != nil {
			fmt.Println("extra tidak valid:", err)
			os.Exit(2)
		}
		for k, vs := range more {
			q.Set(k, vs[0])
		}
	}

	endpoint := "wss://streaming.assemblyai.com/v3/ws?" + q.Encode()
	fmt.Printf("model=%s interval=%d max_speakers=%d\n\n", *model, *interval, *maxSpeakers)

	conn, resp, err := websocket.DefaultDialer.Dial(endpoint,
		http.Header{"Authorization": []string{key}})
	if err != nil {
		status := ""
		if resp != nil {
			status = resp.Status
		}
		fmt.Println("dial gagal:", err, status)
		os.Exit(1)
	}
	defer conn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("\n[tutup]", err)
				return
			}
			var probe struct {
				Type         string `json:"type"`
				TurnOrder    int    `json:"turn_order"`
				EndOfTurn    bool   `json:"end_of_turn"`
				Transcript   string `json:"transcript"`
				SpeakerLabel string `json:"speaker_label"`
				Words        []struct {
					Speaker string `json:"speaker"`
				} `json:"words"`
			}
			_ = json.Unmarshal(raw, &probe)

			switch probe.Type {
			case "Begin", "Termination", "SpeakerRevision":
				fmt.Printf("\n=== %s ===\n%s\n", probe.Type, string(raw))
			case "Turn":
				if !probe.EndOfTurn || probe.Transcript == "" {
					continue
				}
				speakers := map[string]int{}
				for _, w := range probe.Words {
					speakers[w.Speaker]++
				}
				fmt.Printf("TURN %-2d speaker_label=%-8q words_speakers=%v  %.60s\n",
					probe.TurnOrder, probe.SpeakerLabel, speakers, probe.Transcript)
			}
		}
	}()

	// 4096 byte = 128 ms, dikirim real-time seperti mikrofon sungguhan.
	const chunk = 4096
	for i := 0; i < len(pcm); i += chunk {
		end := min(i+chunk, len(pcm))
		if err := conn.WriteMessage(websocket.BinaryMessage, pcm[i:end]); err != nil {
			fmt.Println("kirim gagal:", err)
			break
		}
		time.Sleep(128 * time.Millisecond)
	}

	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"Terminate"}`))
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		fmt.Println("\n[timeout menunggu Termination]")
	}
}
