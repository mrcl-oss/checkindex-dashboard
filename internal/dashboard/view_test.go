package dashboard

import (
	"reflect"
	"testing"

	"github.com/mrcl-oss/checkIndex/report"
)

func TestBuildViewModelSortsDynamicClustersAndUsesClusterTotals(t *testing.T) {
	const largeCount int64 = 5_000_000_000
	const northSecondaryCount int64 = 11
	const northTotal int64 = largeCount + northSecondaryCount
	const southTotal int64 = 19
	reportData := report.Report{
		Collections: map[string]report.Collection{
			"orders": {
				"customer_id": {
					Clusters: map[string]report.Cluster{
						"zeta": {
							Nodes: map[string]report.NodeUsage{
								"zeta-node": {Count: southTotal, Primary: false},
							},
							Total: southTotal,
						},
						"alpha": {
							Nodes: map[string]report.NodeUsage{
								"north-node-z": {Count: northSecondaryCount, Primary: false},
								"north-node-a": {Count: largeCount, Primary: true},
							},
							Total: northTotal,
						},
					},
					Total: northTotal + southTotal,
				},
				"created_at": {
					Clusters: map[string]report.Cluster{
						"zeta": {
							Nodes: map[string]report.NodeUsage{
								"created-node": {Count: 4, Primary: true},
							},
							Total: 4,
						},
					},
					Total: 4,
				},
			},
			"accounts": {
				"username": {
					Clusters: map[string]report.Cluster{
						"zeta": {
							Nodes: map[string]report.NodeUsage{
								"account-node": {Count: 13, Primary: true},
							},
							Total: 13,
						},
						"alpha": {
							Nodes: map[string]report.NodeUsage{
								"username-node": {Count: 7, Primary: false},
							},
							Total: 7,
						},
					},
					Total: 20,
				},
				"email": {
					Clusters: map[string]report.Cluster{
						"alpha": {
							Nodes: map[string]report.NodeUsage{
								"email-node": {Count: 3, Primary: true},
							},
							Total: 3,
						},
					},
					Total: 3,
				},
			},
		},
	}

	view := BuildViewModel(Snapshot{Report: reportData})
	if got, want := clusterNames(view.Clusters), []string{"alpha", "zeta"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("cluster order = %v, want %v", got, want)
	}

	alpha := findClusterView(t, view, "alpha")
	if got, want := collectionNames(alpha.Collections), []string{"accounts", "orders"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("alpha collection order = %v, want %v", got, want)
	}
	accounts := findCollectionView(t, alpha, "accounts")
	if got, want := indexNames(accounts.Indexes), []string{"email", "username"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("alpha/accounts index order = %v, want %v", got, want)
	}

	orders := findCollectionView(t, alpha, "orders")
	customerIndex := findIndexView(t, orders, "customer_id")
	if customerIndex.Total != northTotal {
		t.Fatalf("alpha customer_id total = %d, want cluster total %d", customerIndex.Total, northTotal)
	}
	if customerIndex.Total == reportData.Collections["orders"]["customer_id"].Total {
		t.Fatalf("alpha customer_id total used cross-cluster index total %d", customerIndex.Total)
	}
	wantNodes := []NodeView{
		{Name: "north-node-a", Count: largeCount, Primary: true},
		{Name: "north-node-z", Count: northSecondaryCount, Primary: false},
	}
	if !reflect.DeepEqual(customerIndex.Nodes, wantNodes) {
		t.Fatalf("alpha customer_id nodes = %#v, want %#v", customerIndex.Nodes, wantNodes)
	}

	zeta := findClusterView(t, view, "zeta")
	if got, want := collectionNames(zeta.Collections), []string{"accounts", "orders"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("zeta collection order = %v, want %v", got, want)
	}
	zetaOrders := findCollectionView(t, zeta, "orders")
	if got, want := indexNames(zetaOrders.Indexes), []string{"created_at", "customer_id"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("zeta/orders index order = %v, want %v", got, want)
	}
	zetaCustomerIndex := findIndexView(t, zetaOrders, "customer_id")
	if zetaCustomerIndex.Total != southTotal {
		t.Fatalf("zeta customer_id total = %d, want %d", zetaCustomerIndex.Total, southTotal)
	}
	if got, want := zetaCustomerIndex.Nodes, []NodeView{{Name: "zeta-node", Count: southTotal, Primary: false}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("zeta customer_id nodes = %#v, want %#v", got, want)
	}
}

func clusterNames(clusters []ClusterView) []string {
	names := make([]string, len(clusters))
	for i, cluster := range clusters {
		names[i] = cluster.Name
	}
	return names
}

func collectionNames(collections []CollectionView) []string {
	names := make([]string, len(collections))
	for i, collection := range collections {
		names[i] = collection.Name
	}
	return names
}

func indexNames(indexes []IndexView) []string {
	names := make([]string, len(indexes))
	for i, index := range indexes {
		names[i] = index.Name
	}
	return names
}

func findClusterView(t *testing.T, view ViewModel, name string) ClusterView {
	t.Helper()
	for _, cluster := range view.Clusters {
		if cluster.Name == name {
			return cluster
		}
	}
	t.Fatalf("cluster %q not found in view", name)
	return ClusterView{}
}

func findCollectionView(t *testing.T, cluster ClusterView, name string) CollectionView {
	t.Helper()
	for _, collection := range cluster.Collections {
		if collection.Name == name {
			return collection
		}
	}
	t.Fatalf("collection %q not found in cluster %q", name, cluster.Name)
	return CollectionView{}
}

func findIndexView(t *testing.T, collection CollectionView, name string) IndexView {
	t.Helper()
	for _, index := range collection.Indexes {
		if index.Name == name {
			return index
		}
	}
	t.Fatalf("index %q not found in collection %q", name, collection.Name)
	return IndexView{}
}
