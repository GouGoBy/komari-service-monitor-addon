package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/komari-monitor/komari/database/models"
	"github.com/komari-monitor/komari/internal/availability"
	"github.com/komari-monitor/komari/pkg/rpc"
	"gorm.io/gorm"
)

func init() {
	RegisterWithGroupAndMeta("getServiceMonitors", rpc.RoleAdmin, serviceListAdmin, &rpc.MethodMeta{Name: "admin:getServiceMonitors", Summary: "List service monitors and availability"})
	RegisterWithGroupAndMeta("saveServiceMonitor", rpc.RoleAdmin, serviceSave, &rpc.MethodMeta{Name: "admin:saveServiceMonitor", Summary: "Create or replace a service monitor"})
	RegisterWithGroupAndMeta("deleteServiceMonitor", rpc.RoleAdmin, serviceDelete, &rpc.MethodMeta{Name: "admin:deleteServiceMonitor", Summary: "Delete a service monitor and history"})
	RegisterWithGroupAndMeta("getServiceHistory", rpc.RoleAdmin, serviceHistoryAdmin, &rpc.MethodMeta{Name: "admin:getServiceHistory", Summary: "Get service check history"})
	regPublic("getServiceMonitors", serviceListPublic, "List publicly visible service status")
	regPublic("getServiceHistory", serviceHistoryPublic, "Get public service check history")
}

func serviceListAdmin(context.Context, *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	return serviceList(true)
}
func serviceListPublic(context.Context, *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	return serviceList(false)
}
func serviceList(admin bool) (any, *rpc.JsonRpcError) {
	v, err := availability.List(admin)
	if err != nil {
		return nil, rpc.MakeError(rpc.InternalError, "Failed to read service monitors", nil)
	}
	return v, nil
}

func serviceSave(_ context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	var raw json.RawMessage
	if err := req.BindParams(&raw); err != nil {
		return nil, rpc.MakeError(rpc.InvalidParams, "Invalid monitor", nil)
	}
	var m models.ServiceMonitor
	var secret struct {
		Token *string `json:"token"`
	}
	if json.Unmarshal(raw, &m) != nil || json.Unmarshal(raw, &secret) != nil {
		return nil, rpc.MakeError(rpc.InvalidParams, "Invalid monitor", nil)
	}
	if err := availability.Save(&m, secret.Token); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, rpc.MakeError(rpc.InvalidParams, "Monitor not found", nil)
		}
		return nil, rpc.MakeError(rpc.InvalidParams, "Could not save monitor: "+err.Error(), nil)
	}
	return map[string]any{"id": m.ID}, nil
}

func serviceDelete(_ context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	id, e := serviceID(req)
	if e != nil {
		return nil, e
	}
	if err := availability.Delete(id); err != nil {
		return nil, rpc.MakeError(rpc.InvalidParams, "Could not delete monitor", nil)
	}
	return nil, nil
}

func serviceID(req *rpc.JsonRpcRequest) (uint, *rpc.JsonRpcError) {
	var p struct {
		ID json.Number `json:"id"`
	}
	if err := req.BindParams(&p); err != nil {
		return 0, rpc.MakeError(rpc.InvalidParams, "Positive id is required", nil)
	}
	id, err := strconv.ParseUint(p.ID.String(), 10, strconv.IntSize)
	if err != nil || id == 0 {
		return 0, rpc.MakeError(rpc.InvalidParams, "Positive id is required", nil)
	}
	return uint(id), nil
}

func serviceHistoryAdmin(_ context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	return serviceHistory(req, true)
}
func serviceHistoryPublic(_ context.Context, req *rpc.JsonRpcRequest) (any, *rpc.JsonRpcError) {
	return serviceHistory(req, false)
}
func serviceHistory(req *rpc.JsonRpcRequest, admin bool) (any, *rpc.JsonRpcError) {
	id, e := serviceID(req)
	if e != nil {
		return nil, e
	}
	h, err := availability.History(id, admin)
	if err != nil {
		return nil, rpc.MakeError(rpc.InvalidParams, "Monitor unavailable", nil)
	}
	return h, nil
}
