package memory

import "context"

type ZepClientMock struct {
	AddFactFunc      func(context.Context, Fact) error
	GetFactsFunc     func(context.Context, string, int) ([]Fact, error)
	SearchGraphFunc  func(context.Context, SearchRequest) (SearchResponse, error)
	DeleteNodeFunc   func(context.Context, string) error
	GetGraphDataFunc func(context.Context, string) (GraphData, error)
	DeleteGraphFunc  func(context.Context, string) error
}

func (m ZepClientMock) AddFact(ctx context.Context, fact Fact) error {
	if m.AddFactFunc != nil {
		return m.AddFactFunc(ctx, fact)
	}
	return nil
}

func (m ZepClientMock) GetFacts(ctx context.Context, graphID string, limit int) ([]Fact, error) {
	if m.GetFactsFunc != nil {
		return m.GetFactsFunc(ctx, graphID, limit)
	}
	return nil, nil
}

func (m ZepClientMock) SearchGraph(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	if m.SearchGraphFunc != nil {
		return m.SearchGraphFunc(ctx, req)
	}
	return SearchResponse{}, nil
}

func (m ZepClientMock) DeleteNode(ctx context.Context, nodeID string) error {
	if m.DeleteNodeFunc != nil {
		return m.DeleteNodeFunc(ctx, nodeID)
	}
	return nil
}

func (m ZepClientMock) GetGraphData(ctx context.Context, graphID string) (GraphData, error) {
	if m.GetGraphDataFunc != nil {
		return m.GetGraphDataFunc(ctx, graphID)
	}
	return GraphData{}, nil
}

func (m ZepClientMock) DeleteGraph(ctx context.Context, graphID string) error {
	if m.DeleteGraphFunc != nil {
		return m.DeleteGraphFunc(ctx, graphID)
	}
	return nil
}
