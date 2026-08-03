package dto

type PlayGroundRequest struct {
	Model      string `json:"model,omitempty"`
	RouteGroup string `json:"route_group,omitempty"`
	Group      string `json:"group,omitempty"`
}
