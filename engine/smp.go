package engine

import (
	"sync"
	"sync/atomic"
)

// Threads is the number of search goroutines SearchLazySMP spawns (1 =
// classic single-threaded search, no locking overhead anywhere -- see
// ttSMPActive in tt.go). Set via the UCI "Threads" option, defaulting to 1
// so existing single-threaded behavior is unchanged unless a GUI/tester
// opts in.
var Threads = 1

// SearchLazySMP runs Threads-1 helper searches alongside a main search, all
// against the same position and all sharing the package-level TT (see
// tt.go). Helpers get no dedicated time check of their own beyond the
// shared stop signal below -- they run the same iterative-deepening loop as
// the main search and simply get interrupted once the main search
// concludes.
//
// The result returned is always the main search's own best move/score:
// helpers exist only to seed the shared TT with additional
// depth/best-move/bound information (found via naturally-diverging
// goroutine scheduling -- no explicit diversification of helper searches is
// needed), not to be trusted for the final answer themselves. This is the
// standard "Lazy SMP" design (as opposed to a proper split-point/YBWC
// parallel search): dead simple, no work-stealing or synchronization beyond
// the TT, and known to scale sub-linearly but positively up to a moderate
// thread count.
func SearchLazySMP(main *Search) (int32, Move) {
	if Threads <= 1 {
		return main.Search()
	}

	var stop int32
	main.SetStopSignal(&stop)

	atomic.StoreInt32(&ttSMPActive, 1)
	defer atomic.StoreInt32(&ttSMPActive, 0)

	var wg sync.WaitGroup
	for i := 1; i < Threads; i++ {
		helper := &Search{
			MaxDepth:  main.MaxDepth,
			TimeLimit: main.TimeLimit,
			silent:    true,
		}
		helper.Init(&main.Pos)
		helper.SetStopSignal(&stop)

		wg.Add(1)
		go func() {
			defer wg.Done()
			helper.Search()
		}()
	}

	score, move := main.Search()

	atomic.StoreInt32(&stop, 1)
	wg.Wait()

	return score, move
}
