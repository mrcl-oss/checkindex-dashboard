package dashboard

import (
	"sort"
	"time"
)

// ViewModel is the sorted, presentation-ready dashboard snapshot.
type ViewModel struct {
	Clusters    []ClusterView
	Partial     bool
	Failed      bool
	Stale       bool
	LastAttempt *TimestampView
	LastSuccess *TimestampView
}

// ClusterView contains the collections and indexes observed for one cluster.
type ClusterView struct {
	Name        string
	Collections []CollectionView
}

// CollectionView contains the indexes observed for one collection in a cluster.
type CollectionView struct {
	Name    string
	Indexes []IndexView
}

// IndexView contains one cluster's total and only that cluster's observed nodes.
type IndexView struct {
	Name  string
	Total int64
	Nodes []NodeView
}

// NodeView contains one observed node's count and primary role.
type NodeView struct {
	Name    string
	Count   int64
	Primary bool
}

// TimestampView contains machine-readable and human-readable UTC timestamps.
type TimestampView struct {
	DateTime string
	Display  string
}

// BuildViewModel pivots a report into sorted cluster, collection, index, and node sections.
func BuildViewModel(snapshot Snapshot) ViewModel {
	view := ViewModel{
		Partial:     snapshot.Partial || snapshot.Report.Partial,
		Failed:      snapshot.Failed,
		Stale:       snapshot.Stale,
		LastAttempt: timestampView(snapshot.LastAttempt),
		LastSuccess: timestampView(snapshot.LastSuccess),
	}

	clusters := make(map[string]*ClusterView)
	collections := make(map[string]map[string]*CollectionView)
	for collectionName, indexes := range snapshot.Report.Collections {
		for indexName, index := range indexes {
			for clusterName, cluster := range index.Clusters {
				clusterView := clusters[clusterName]
				if clusterView == nil {
					clusterView = &ClusterView{Name: clusterName}
					clusters[clusterName] = clusterView
					collections[clusterName] = make(map[string]*CollectionView)
				}

				collectionView := collections[clusterName][collectionName]
				if collectionView == nil {
					collectionView = &CollectionView{Name: collectionName}
					collections[clusterName][collectionName] = collectionView
				}

				indexView := IndexView{Name: indexName, Total: cluster.Total}
				for nodeName, usage := range cluster.Nodes {
					indexView.Nodes = append(indexView.Nodes, NodeView{
						Name:    nodeName,
						Count:   usage.Count,
						Primary: usage.Primary,
					})
				}
				sort.Slice(indexView.Nodes, func(i, j int) bool {
					return indexView.Nodes[i].Name < indexView.Nodes[j].Name
				})
				collectionView.Indexes = append(collectionView.Indexes, indexView)
			}
		}
	}

	for clusterName, clusterView := range clusters {
		for _, collectionView := range collections[clusterName] {
			sort.Slice(collectionView.Indexes, func(i, j int) bool {
				return collectionView.Indexes[i].Name < collectionView.Indexes[j].Name
			})
			clusterView.Collections = append(clusterView.Collections, *collectionView)
		}
		sort.Slice(clusterView.Collections, func(i, j int) bool {
			return clusterView.Collections[i].Name < clusterView.Collections[j].Name
		})
		view.Clusters = append(view.Clusters, *clusterView)
	}
	sort.Slice(view.Clusters, func(i, j int) bool {
		return view.Clusters[i].Name < view.Clusters[j].Name
	})
	return view
}

func timestampView(value time.Time) *TimestampView {
	if value.IsZero() {
		return nil
	}
	utc := value.UTC()
	return &TimestampView{
		DateTime: utc.Format(time.RFC3339),
		Display:  utc.Format("2006-01-02 15:04:05 UTC"),
	}
}
