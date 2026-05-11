package engine

import (
	"encoding/json"
	"fmt"
)

type FlowNode struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Data     map[string]any `json:"data"`
	Position map[string]any `json:"position"`
}

type FlowEdge struct {
	ID           string         `json:"id"`
	Source       string         `json:"source"`
	Target       string         `json:"target"`
	SourceHandle *string        `json:"sourceHandle"`
	TargetHandle *string        `json:"targetHandle"`
	Data         map[string]any `json:"data"`
}

type Graph struct {
	Nodes       []FlowNode
	Edges       []FlowEdge
	NodeMap     map[string]FlowNode
	Preds       map[string][]string
	Succs       map[string][]string
	EdgeLabels  map[string]map[string]string // source -> target -> label (from edge.data.label)
	StartNodeID string
}

func ParseGraph(nodesJSON, edgesJSON []byte) (*Graph, error) {
	var nodes []FlowNode
	var edges []FlowEdge
	if err := json.Unmarshal(nodesJSON, &nodes); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(edgesJSON, &edges); err != nil {
		return nil, err
	}
	g := &Graph{
		Nodes:      nodes,
		Edges:      edges,
		NodeMap:    map[string]FlowNode{},
		Preds:      map[string][]string{},
		Succs:      map[string][]string{},
		EdgeLabels: map[string]map[string]string{},
	}
	for _, n := range nodes {
		g.NodeMap[n.ID] = n
		if n.Type == "start" {
			if g.StartNodeID != "" {
				return nil, fmt.Errorf("multiple start nodes")
			}
			g.StartNodeID = n.ID
		}
	}
	if g.StartNodeID == "" {
		return nil, fmt.Errorf("missing start node")
	}
	for _, e := range edges {
		g.Preds[e.Target] = append(g.Preds[e.Target], e.Source)
		g.Succs[e.Source] = append(g.Succs[e.Source], e.Target)
		lbl := ""
		if e.Data != nil {
			if v, ok := e.Data["label"].(string); ok {
				lbl = v
			}
		}
		if lbl != "" {
			if g.EdgeLabels[e.Source] == nil {
				g.EdgeLabels[e.Source] = map[string]string{}
			}
			g.EdgeLabels[e.Source][e.Target] = lbl
		}
	}
	return g, nil
}

func (g *Graph) PredCount() map[string]int {
	pc := map[string]int{}
	for id := range g.NodeMap {
		pc[id] = len(g.Preds[id])
	}
	return pc
}
