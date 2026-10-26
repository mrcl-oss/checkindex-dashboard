package dashboard

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/mrcl-oss/checkIndex/report"
)

const DefaultRefreshInterval = time.Minute

// Collector obtains one report snapshot.
type Collector func(context.Context) (report.Report, error)

// Snapshot contains the latest report and collection status metadata.
type Snapshot struct {
	Report      report.Report
	LastAttempt time.Time
	LastSuccess time.Time
	LastError   string
	Failed      bool
	Partial     bool
	Stale       bool
}

// Service periodically collects reports and exposes a concurrency-safe snapshot.
type Service struct {
	collector Collector
	interval  time.Duration

	refreshMu sync.Mutex
	mu        sync.RWMutex
	snapshot  Snapshot
}

// NewService creates a service that refreshes at interval, defaulting to one minute.
// A nil collector uses the public report API to collect all configured profiles.
func NewService(collector Collector, interval time.Duration) *Service {
	if collector == nil {
		collector = func(ctx context.Context) (report.Report, error) {
			return report.Collect(ctx, report.Options{All: true})
		}
	}
	if interval <= 0 {
		interval = DefaultRefreshInterval
	}
	return &Service{collector: collector, interval: interval}
}

// Refresh collects once. Concurrent calls are serialized so collectors never overlap.
func (s *Service) Refresh(ctx context.Context) error {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	attemptedAt := time.Now()
	s.mu.Lock()
	s.snapshot.LastAttempt = attemptedAt
	s.mu.Unlock()

	var collected report.Report
	var err error
	if s.collector == nil {
		err = errors.New("dashboard: collector is nil")
	} else {
		collected, err = s.collector(ctx)
	}

	completedAt := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.snapshot.Report = cloneReport(collected)
		s.snapshot.LastSuccess = completedAt
		s.snapshot.LastError = ""
		s.snapshot.Failed = false
		s.snapshot.Partial = collected.Partial
		s.snapshot.Stale = false
		return nil
	}

	s.snapshot.LastError = err.Error()
	s.snapshot.Failed = true
	s.snapshot.Partial = collected.Partial
	if hasObservations(collected) {
		s.snapshot.Report = cloneReport(collected)
		s.snapshot.Stale = false
	} else {
		s.snapshot.Stale = true
	}
	return err
}

// Snapshot returns an independent copy of the current report and its status.
func (s *Service) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := s.snapshot
	snapshot.Report = cloneReport(snapshot.Report)
	return snapshot
}

// Run refreshes immediately and then at the configured interval until ctx is canceled.
func (s *Service) Run(ctx context.Context) {
	interval := s.interval
	if interval <= 0 {
		interval = DefaultRefreshInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return
		}
		_ = s.Refresh(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func hasObservations(r report.Report) bool {
	for _, indexes := range r.Collections {
		for _, index := range indexes {
			for _, cluster := range index.Clusters {
				if len(cluster.Nodes) > 0 {
					return true
				}
			}
		}
	}
	return false
}

func cloneReport(r report.Report) report.Report {
	if r.Collections == nil {
		return r
	}

	collections := make(map[string]report.Collection, len(r.Collections))
	for collectionName, indexes := range r.Collections {
		if indexes == nil {
			collections[collectionName] = nil
			continue
		}

		clonedIndexes := make(report.Collection, len(indexes))
		for indexName, index := range indexes {
			if index.Clusters != nil {
				clonedClusters := make(map[string]report.Cluster, len(index.Clusters))
				for clusterName, cluster := range index.Clusters {
					if cluster.Nodes != nil {
						clonedNodes := make(map[string]report.NodeUsage, len(cluster.Nodes))
						for nodeName, usage := range cluster.Nodes {
							clonedNodes[nodeName] = usage
						}
						cluster.Nodes = clonedNodes
					}
					clonedClusters[clusterName] = cluster
				}
				index.Clusters = clonedClusters
			}
			clonedIndexes[indexName] = index
		}
		collections[collectionName] = clonedIndexes
	}
	r.Collections = collections
	return r
}
