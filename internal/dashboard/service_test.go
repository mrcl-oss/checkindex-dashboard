package dashboard

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mrcl-oss/checkIndex/report"
)

func TestRefreshPreservesClusterAwarePartialReportReturnedWithError(t *testing.T) {
	wantErr := errors.New("one node was unavailable")
	wantReport := syntheticReport(true)
	service := NewService(func(context.Context) (report.Report, error) {
		return wantReport, wantErr
	}, time.Minute)

	if err := service.Refresh(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Refresh() error = %v, want %v", err, wantErr)
	}

	got := service.Snapshot()
	assertReportEqual(t, got.Report, wantReport)
	if !got.Partial || !got.Report.Partial {
		t.Fatalf("partial status was not preserved: snapshot=%t report=%t", got.Partial, got.Report.Partial)
	}
	if !got.Failed || got.LastError != wantErr.Error() {
		t.Fatalf("failure metadata = failed:%t error:%q, want failed with %q", got.Failed, got.LastError, wantErr)
	}
	if got.Stale {
		t.Fatal("partial data from the latest attempt must not be marked stale")
	}
	if got.LastAttempt.IsZero() {
		t.Fatal("LastAttempt was not recorded")
	}
	if !got.LastSuccess.IsZero() {
		t.Fatalf("LastSuccess = %v, want zero after a failed collection", got.LastSuccess)
	}
}

func TestRefreshRecordsSuccessfulSnapshot(t *testing.T) {
	wantReport := syntheticReport(false)
	service := NewService(func(context.Context) (report.Report, error) {
		return wantReport, nil
	}, time.Minute)

	if err := service.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	got := service.Snapshot()
	if got.LastAttempt.IsZero() || got.LastSuccess.IsZero() {
		t.Fatalf("successful timestamps were not recorded: attempt=%v success=%v", got.LastAttempt, got.LastSuccess)
	}
	if got.Failed || got.Partial || got.Stale || got.LastError != "" {
		t.Fatalf("successful snapshot has failure status: %+v", got)
	}
	assertReportEqual(t, got.Report, wantReport)
}

func TestRefreshRetainsUsefulReportAndMarksItStaleAfterEmptyFailure(t *testing.T) {
	wantErr := errors.New("collection failed")
	attempt := 0
	wantReport := syntheticReport(false)
	service := NewService(func(context.Context) (report.Report, error) {
		attempt++
		if attempt == 1 {
			return wantReport, nil
		}
		return report.Report{Partial: true}, wantErr
	}, time.Minute)

	if err := service.Refresh(context.Background()); err != nil {
		t.Fatalf("initial Refresh() error = %v", err)
	}
	lastSuccess := service.Snapshot().LastSuccess
	if err := service.Refresh(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("failed Refresh() error = %v, want %v", err, wantErr)
	}

	got := service.Snapshot()
	assertReportEqual(t, got.Report, wantReport)
	if !got.Stale || !got.Failed || !got.Partial || got.LastError != wantErr.Error() {
		t.Fatalf("failed snapshot status = %+v", got)
	}
	if !got.LastSuccess.Equal(lastSuccess) {
		t.Fatalf("LastSuccess = %v, want unchanged %v", got.LastSuccess, lastSuccess)
	}
}

func TestSnapshotReturnsIndependentClusterAndNodeMaps(t *testing.T) {
	collected := syntheticReport(false)
	wantReport := syntheticReport(false)
	service := NewService(func(context.Context) (report.Report, error) {
		return collected, nil
	}, time.Minute)
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	first := service.Snapshot()
	mutateSyntheticReport(&first.Report)
	mutateSyntheticReport(&collected)
	assertReportEqual(t, service.Snapshot().Report, wantReport)
}

func TestConcurrentRefreshesDoNotOverlapCollectors(t *testing.T) {
	var active, maximum, calls int32
	entered := make(chan struct{}, 3)
	release := make(chan struct{})
	collector := func(context.Context) (report.Report, error) {
		current := atomic.AddInt32(&active, 1)
		for previous := atomic.LoadInt32(&maximum); current > previous; previous = atomic.LoadInt32(&maximum) {
			if atomic.CompareAndSwapInt32(&maximum, previous, current) {
				break
			}
		}
		atomic.AddInt32(&calls, 1)
		entered <- struct{}{}
		<-release
		atomic.AddInt32(&active, -1)
		return report.Report{}, nil
	}
	service := NewService(collector, time.Minute)

	done := make(chan struct{}, 3)
	go func() {
		_ = service.Refresh(context.Background())
		done <- struct{}{}
	}()
	<-entered
	started := make(chan struct{}, 2)
	for range 2 {
		go func() {
			started <- struct{}{}
			_ = service.Refresh(context.Background())
			done <- struct{}{}
		}()
	}
	<-started
	<-started

	overlapped := false
	select {
	case <-entered:
		overlapped = true
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	for range 3 {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("Refresh() did not complete")
		}
	}
	if overlapped || atomic.LoadInt32(&maximum) != 1 || atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("collector concurrency = %d, calls = %d, overlapped = %t; want 1, 3, false", atomic.LoadInt32(&maximum), atomic.LoadInt32(&calls), overlapped)
	}
}

func TestRunRefreshesUntilContextIsCanceled(t *testing.T) {
	collected := make(chan struct{}, 100)
	service := NewService(func(context.Context) (report.Report, error) {
		collected <- struct{}{}
		return report.Report{}, nil
	}, time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		service.Run(ctx)
		close(done)
	}()

	for range 2 {
		select {
		case <-collected:
		case <-time.After(time.Second):
			cancel()
			t.Fatal("Run() did not perform periodic refreshes")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run() did not stop after context cancellation")
	}
}

func syntheticReport(partial bool) report.Report {
	const (
		northPrimaryCount   int64 = 5_000_000_000
		northSecondaryCount int64 = 11
		southPrimaryCount   int64 = 17
		southSecondaryCount int64 = 23
	)
	northTotal := northPrimaryCount + northSecondaryCount
	southTotal := southPrimaryCount + southSecondaryCount
	return report.Report{
		Collections: map[string]report.Collection{
			"orders": {
				"customer_id": {
					Clusters: map[string]report.Cluster{
						"cluster-north": {
							Nodes: map[string]report.NodeUsage{
								"north-node-a": {Count: northPrimaryCount, Primary: true},
								"north-node-b": {Count: northSecondaryCount, Primary: false},
							},
							Total: northTotal,
						},
						"cluster-south": {
							Nodes: map[string]report.NodeUsage{
								"south-node-a": {Count: southPrimaryCount, Primary: true},
								"south-node-b": {Count: southSecondaryCount, Primary: false},
							},
							Total: southTotal,
						},
					},
					Total: northTotal + southTotal,
				},
			},
		},
		Partial: partial,
	}
}

func mutateSyntheticReport(r *report.Report) {
	collection := r.Collections["orders"]
	index := collection["customer_id"]
	cluster := index.Clusters["cluster-north"]
	cluster.Nodes["north-node-a"] = report.NodeUsage{Count: 1, Primary: false}
	cluster.Total = -1
	index.Clusters["cluster-north"] = cluster
	index.Total = -2
	collection["customer_id"] = index
	r.Collections["orders"] = collection
}

func assertReportEqual(t *testing.T, got, want report.Report) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("report mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}
