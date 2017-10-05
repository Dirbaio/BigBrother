package main

import (
	"os"
	"log"
	"syscall"
	"time"
	"errors"
	"fmt"
)

type Janitor struct {
	done chan struct{}
}

func NewJanitor() *Janitor {
	return &Janitor{
		done: make(chan struct{}),
	}
}

func (this *Janitor) Run() {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			this.tick()
		case <-this.done:
			return
		}
	}
}

func (this *Janitor) Stop() {
	close(this.done)
}

func (this *Janitor) tick() {
	// TODO delete empty periods

	for {
		free, err := this.getFreeSpace()
		if err != nil {
			log.Printf("Janitor: Error stating free space: %v\n", err)
			return
		}
		log.Printf("Janitor: free space: %f\n", free)

		// TODO make this configurable
		// Erase stuff until there's more than 2% free.
		if free > 0.02 {
			return
		}

		err = this.deleteOldSegments()
		if err != nil {
			log.Printf("Janitor: Error deleting old segments: %v\n", err)
			return
		}
	}
}

func (this *Janitor) getFreeSpace() (float64, error) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(storagePath, &fs)
	if err != nil {
		return 0, err
	}

	free := float64(fs.Bavail) / float64(fs.Blocks)
	return free, nil
}

func (this *Janitor) deleteOldSegments() error {
	// Fetch 10 segments to delete in bulk.
	segments := []Segment{}
	err := db.Select(&segments, "DELETE FROM segment WHERE ctid IN (SELECT ctid FROM segment ORDER BY time ASC limit 10) RETURNING *")
	if err != nil {
		log.Fatal(err)
	}
	if len(segments) == 0 {
		return errors.New("No segments to delete!")
	}

	for _, s := range segments {
		path := fmt.Sprintf("%s/%d/%d/chunk-stream0-%05d.m4s", storagePath, s.CameraID, s.PeriodID, s.Index)
		log.Printf("Janitor: deleting %s\n", path)
		err = os.Remove(path)
		if err != nil {
			log.Printf("Janitor: Error deleting %s: %v\n", path, err)
		}
	}

	return nil
}
