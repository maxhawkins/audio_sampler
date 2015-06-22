package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/krig/go-sox"
)

func main() {
	var (
		dir         = flag.String("dir", ".", "input directory")
		outPath     = flag.String("out", "out.wav", "output filename")
		chunkLength = flag.Float64("length", 1, "length of chunk in seconds")
		chunkCount  = flag.Int("count", 10, "number of chunks")
	)
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	if !sox.Init() {
		log.Fatal("Failed to initialize SoX")
	}
	defer sox.Quit()

	var corpus Corpus
	err := filepath.Walk(*dir, func(path string, _ os.FileInfo, _ error) error {
		corpus.AddPath(path)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	signal := sox.NewSignalInfo(44100, 2, 16, 1e8, nil)
	out := sox.OpenWrite(*outPath, signal, nil, "wav")
	if out == nil {
		log.Fatal("Failed to open memory buffer")
	}

	for i := 0; i < *chunkCount; i++ {
		path, offsetSec := corpus.Sample()

		in := sox.OpenRead(path)
		if in == nil {
			log.Fatal("failed to open input file")
		}
		defer in.Release()

		start := offsetSec
		end := start + *chunkLength

		fmt.Println("adding", path, start, end)

		CopyClip(out, in, start, end)
	}
}

type Corpus struct {
	paths     []string
	durations []float64
	total     float64
}

func GetDuration(path string) (float64, error) {
	in := sox.OpenRead(path)
	if in == nil {
		return 0, fmt.Errorf("failed to open input file")
	}
	defer in.Release()

	channels := in.Signal().Channels()
	if channels < 1 {
		return 0, fmt.Errorf("can't read channel count")
	}
	length := in.Signal().Length()
	if length < 1 {
		return 0, fmt.Errorf("can't read file length")
	}
	rate := in.Signal().Rate()
	if rate < 1 {
		return 0, fmt.Errorf("can't read file rate")
	}

	if !in.Seekable() {
		return 0, fmt.Errorf("not seekable")
	}

	duration := float64(length) / float64(channels) / rate

	return duration, nil
}

func (c *Corpus) AddPath(path string) error {
	duration, err := GetDuration(path)
	if err != nil {
		return err
	}

	c.paths = append(c.paths, path)
	c.durations = append(c.durations, duration)
	c.total += duration

	return nil
}

func (c *Corpus) Sample() (path string, offsetSec float64) {
	var selection int
	offsetSec = rand.Float64() * c.total
	for i, dur := range c.durations {
		if offsetSec < dur {
			selection = i
			break
		}
		offsetSec -= dur
	}

	return c.paths[selection], offsetSec
}

func CopyClip(dst, src *sox.Format, start, end float64) {
	chain := sox.CreateEffectsChain(src.Encoding(), dst.Encoding())
	defer chain.Release()

	signal := src.Signal().Copy()

	e := sox.CreateEffect(sox.FindEffect("input"))
	e.Options(src)
	chain.Add(e, signal, src.Signal())
	e.Release()

	e = sox.CreateEffect(sox.FindEffect("trim"))
	e.Options(fmt.Sprint(start), "="+fmt.Sprint(end))
	chain.Add(e, signal, src.Signal())
	e.Release()

	if src.Signal().Rate() != dst.Signal().Rate() {
		e = sox.CreateEffect(sox.FindEffect("rate"))
		e.Options()
		chain.Add(e, signal, dst.Signal())
		e.Release()
	}

	if src.Signal().Channels() != dst.Signal().Channels() {
		e = sox.CreateEffect(sox.FindEffect("channels"))
		e.Options()
		chain.Add(e, signal, dst.Signal())
		e.Release()
	}

	e = sox.CreateEffect(sox.FindEffect("output"))
	e.Options(dst)
	chain.Add(e, signal, dst.Signal())
	e.Release()

	chain.Flow()
}
