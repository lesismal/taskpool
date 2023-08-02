package taskpool

import "sync"

type Pool struct {
	closed bool

	cond      sync.Cond
	queue     []func()
	queueSize int
	waiting   int

	concurrency    int
	maxConcurrency int
}

func (p *Pool) Stop() {
	p.closed = true
	p.cond.Broadcast()
}

func (p *Pool) Go(f func()) {
again:
	if p.closed {
		return
	}

	mux := p.cond.L
	mux.Lock()

	if p.concurrency+1 < p.maxConcurrency {
		p.concurrency++
		mux.Unlock()
		go func() {
			f()
			for {
				mux.Lock()
				if len(p.queue) > 0 {
					f = p.queue[0]
					p.queue = p.queue[1:]
					if p.waiting > 0 {
						p.cond.Signal()
					}
					mux.Unlock()
					f()
					continue
				}
				p.concurrency--
				mux.Unlock()
				return
			}
		}()
		return
	}
	if len(p.queue) < p.queueSize {
		p.queue = append(p.queue, f)
		mux.Unlock()
		return
	}

	p.waiting++
	p.cond.Wait()
	p.waiting--
	mux.Unlock()
	goto again
}

func NewPool(concurrency, queueSize int) *Pool {
	return &Pool{cond: sync.Cond{L: &sync.Mutex{}}, maxConcurrency: concurrency, queueSize: queueSize}
}
