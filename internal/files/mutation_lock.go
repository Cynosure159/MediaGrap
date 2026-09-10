package files

import "context"

// Shared by browser and automation writers. A single application process uses
// one bounded mutation lane, so independently created plans cannot interleave.
var mutationLane = make(chan struct{}, 1)

func LockMutation(ctx context.Context) (func(), error) {
	select {
	case mutationLane <- struct{}{}:
		return func() { <-mutationLane }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
