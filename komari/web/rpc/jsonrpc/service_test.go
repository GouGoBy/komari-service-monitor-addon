package jsonrpc

import (
	"github.com/komari-monitor/komari/pkg/rpc"
	"testing"
)

func TestServiceIDAcceptsHTTPQueryAndRPCNumber(t *testing.T) {
	for _, value := range []any{"12", float64(12), uint(12)} {
		id, e := serviceID(&rpc.JsonRpcRequest{Params: map[string]any{"id": value}})
		if e != nil || id != 12 {
			t.Fatalf("value %v: %d %v", value, id, e)
		}
	}
	id, e := serviceID(&rpc.JsonRpcRequest{Params: map[string]any{"id": float64(10000000)}})
	if e != nil || id != 10000000 {
		t.Fatalf("large ID: %d %v", id, e)
	}
	for _, value := range []any{"0", "-1", "1.5", "abc", nil} {
		if _, e := serviceID(&rpc.JsonRpcRequest{Params: map[string]any{"id": value}}); e == nil {
			t.Fatalf("accepted %v", value)
		}
	}
}
