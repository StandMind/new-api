package dto

type PlayGroundRequest struct {
	Model           string `json:"model,omitempty"`
	RouteGroup      string `json:"route_group,omitempty"`
	RoutingPriority string `json:"routing_priority,omitempty"`
	Group           string `json:"group,omitempty"`
}
