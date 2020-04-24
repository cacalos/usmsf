package utils

import (
	"io"
	"net/http"
	"sync"
	"time"
)

// writeFlusher HTTP Writer & Flush 인터페이스
type writeFlusher interface {
	io.Writer
	http.Flusher
}

// maxLatencyWriter 최대 지연 시간이 정해 지연되면 Flush를
type maxLatencyWriter struct {
	dst     writeFlusher  // Flush가 가능한 Writer
	latency time.Duration // 송신 지연 시간

	lk              sync.Mutex // protects Write + Flush
	done            chan bool  // 종료 체절
	onExitFlushLoop func()     //Flush 종료시 호출될 Callback 함수
}

var bufferPool *sync.Pool
var bufferPoolOnce sync.Once

///////////////////////////////////////////////////////////////////////////////
// maxLatencyWirter 클래스 함수들
///////////////////////////////////////////////////////////////////////////////

// Write 전달된 데이터를를 Write 한다.
func (m *maxLatencyWriter) Write(p []byte) (int, error) {
	m.lk.Lock()
	defer m.lk.Unlock()
	return m.dst.Write(p)
}

// flushLoop 설정된 주기로 데이터가 송출되도록 헤당 Writer에 Flush를 호출한다.
func (m *maxLatencyWriter) flushLoop() {
	t := time.NewTicker(m.latency)
	defer t.Stop()
	for {
		select {
		case <-m.done:
			if m.onExitFlushLoop != nil {
				m.onExitFlushLoop()
			}
			return
		case <-t.C:
			m.lk.Lock()
			m.dst.Flush()
			m.lk.Unlock()
		}
	}
}

// stop 해당
func (m *maxLatencyWriter) stop() {
	m.done <- true
}

///////////////////////////////////////////////////////////////////////////////
// HTTP 관련 Utility 함수들
///////////////////////////////////////////////////////////////////////////////

// GetIOBufferPool Buffer Pool을 가져온다.
func GetIOBufferPool() *sync.Pool {
	bufferPoolOnce.Do(func() {
		makeBuffer := func() interface{} { return make([]byte, 0, 32*1024) }
		bufferPool = &sync.Pool{New: makeBuffer}
	})
	return bufferPool
}

// NewIOBuffer 새로운 IO Buffer를 생성한다.
func NewIOBuffer() []byte {
	return GetIOBufferPool().Get().([]byte)
}

// DestroyIOBuffer 해당 IO Buffer를 소멸한다.
func DestroyIOBuffer(buf []byte) {
	GetIOBufferPool().Put(buf)
}

// CopyBufferWithFlush Reader로 부터 Writer로 버퍼를 복사한다.
// 단. 지연 시간동안 Writer 데이터가 송신되지 않으면 강제로 Writer의 Flush가 호출 된다.
func CopyBufferWithFlush(dst io.Writer, src io.Reader, buf []byte, flushInterval time.Duration) (written int64, err error) {
	if flushInterval != 0 {
		if wf, ok := dst.(writeFlusher); ok {
			mlw := &maxLatencyWriter{
				dst:     wf,
				latency: flushInterval,
				done:    make(chan bool),
			}
			go mlw.flushLoop()
			defer mlw.stop()
			dst = mlw
		}
	}
	bufCap := cap(buf)
	return io.CopyBuffer(dst, src, buf[0:bufCap:bufCap])
}

// CopyIOWithFlush 전달된 소스의 메시지 Body로 부터 대상 메시지 Writer의 Body를 복사한다.
func CopyIOWithFlush(dst io.Writer, src io.Reader, flushInterval time.Duration) (written int64, err error) {
	buf := NewIOBuffer()
	defer DestroyIOBuffer(buf)
	return CopyBufferWithFlush(dst, src, buf, flushInterval)
}
