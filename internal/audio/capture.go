package audio

import (
	"context"
	"fmt"
	"log"

	"github.com/jfreymuth/pulse"
)

// Capturer records PCM audio from a PulseAudio source and sends chunks on a channel.
type Capturer struct {
	source     string // PulseAudio source name (empty = default)
	sampleRate int
	chunkMs    int
	pcmCh      chan<- []byte
}

func NewCapturer(source string, sampleRate, chunkMs int, pcmCh chan<- []byte) *Capturer {
	return &Capturer{
		source:     source,
		sampleRate: sampleRate,
		chunkMs:    chunkMs,
		pcmCh:      pcmCh,
	}
}

// Run connects to PulseAudio and streams audio until ctx is cancelled.
func (c *Capturer) Run(ctx context.Context) error {
	client, err := pulse.NewClient(pulse.ClientApplicationName("local-stt"))
	if err != nil {
		return fmt.Errorf("pulse connect: %w", err)
	}
	defer client.Close()

	var opts []pulse.RecordOption
	if c.source != "" {
		// Look up the named source
		sources, err := client.ListSources()
		if err != nil {
			return fmt.Errorf("list sources: %w", err)
		}
		for _, src := range sources {
			if src.Name() == c.source {
				opts = append(opts, pulse.RecordSource(src))
				break
			}
		}
	}
	opts = append(opts, pulse.RecordSampleRate(c.sampleRate))
	opts = append(opts, pulse.RecordMono)

	// Use an Int16Writer that sends PCM chunks to the channel
	writer := pulse.Int16Writer(func(buf []int16) (int, error) {
		raw := int16ToBytes(buf)
		select {
		case c.pcmCh <- raw:
		default:
			// Drop frame if channel is full
		}
		return len(buf), nil
	})

	stream, err := client.NewRecord(writer, opts...)
	if err != nil {
		return fmt.Errorf("pulse record: %w", err)
	}

	stream.Start()
	log.Printf("audio: recording from %q at %dHz mono", c.source, c.sampleRate)

	<-ctx.Done()
	stream.Stop()
	return nil
}

// int16ToBytes converts int16 samples to little-endian byte slice.
func int16ToBytes(samples []int16) []byte {
	out := make([]byte, len(samples)*2)
	for i, s := range samples {
		out[i*2] = byte(s)
		out[i*2+1] = byte(s >> 8)
	}
	return out
}
