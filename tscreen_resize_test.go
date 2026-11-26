//go:build !windows

package tcell

import (
	"io"
	"os"
	"sync"
	"testing"
	"time"
)

type resizeTty struct {
	sync.Mutex
	w, h int
	stop chan struct{}
}

func (t *resizeTty) Start() error {
	t.Lock()
	t.stop = make(chan struct{})
	t.Unlock()
	return nil
}

func (t *resizeTty) Drain() error {
	t.Lock()
	defer t.Unlock()
	select {
	case <-t.stop:
	default:
		close(t.stop)
	}
	return nil
}

func (t *resizeTty) Stop() error            { return nil }
func (t *resizeTty) Close() error           { return nil }
func (t *resizeTty) NotifyResize(cb func()) {}
func (t *resizeTty) Write(b []byte) (int, error) {
	return len(b), nil
}

func (t *resizeTty) Read(b []byte) (int, error) {
	t.Lock()
	stop := t.stop
	t.Unlock()
	<-stop
	return 0, io.EOF
}

func (t *resizeTty) WindowSize() (int, int, error) {
	t.Lock()
	defer t.Unlock()
	return t.w, t.h, nil
}

func (t *resizeTty) setSize(w, h int) {
	t.Lock()
	t.w, t.h = w, h
	t.Unlock()
}

func TestResumeAfterWindowSizeChangedWhileSuspended(t *testing.T) {
	t.Setenv("TERM", "xterm")
	_ = os.Unsetenv("COLORTERM")
	tty := &resizeTty{w: 80, h: 24}
	s, err := NewTerminfoScreenFromTty(tty)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	s.Show()

	if err := s.Suspend(); err != nil {
		t.Fatal(err)
	}
	tty.setSize(79, 24)
	if err := s.Resume(); err != nil {
		t.Fatal(err)
	}
	tty.setSize(80, 24)

	done := make(chan struct{})
	go func() {
		s.SetContent(0, 0, 'x', nil, StyleDefault)
		s.Show()
		close(done)
	}()
	select {
	case <-done:
		s.Fini()
	case <-time.After(2 * time.Second):
		t.Fatal("Show did not return after the window size changed while suspended")
	}
}
