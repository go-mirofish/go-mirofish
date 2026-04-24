package simulation

type CreateResponse struct {
	SimulationID  string `json:"simulation_id"`
	ProjectID     string `json:"project_id"`
	GraphID       string `json:"graph_id"`
	Status        string `json:"status"`
	EnableTwitter bool   `json:"enable_twitter"`
	EnableReddit  bool   `json:"enable_reddit"`
	CreatedAt     string `json:"created_at"`
}

type CreateRequest struct {
	ProjectID     string `json:"project_id"`
	GraphID       string `json:"graph_id,omitempty"`
	EnableTwitter *bool  `json:"enable_twitter,omitempty"`
	EnableReddit  *bool  `json:"enable_reddit,omitempty"`
}

type DeleteResponse struct {
	SimulationID string   `json:"simulation_id"`
	Warnings     []string `json:"warnings"`
}

type Action struct {
	RoundNum   int            `json:"round_num"`
	Timestamp  string         `json:"timestamp"`
	Platform   string         `json:"platform"`
	AgentID    int            `json:"agent_id"`
	AgentName  string         `json:"agent_name"`
	ActionType string         `json:"action_type"`
	ActionArgs map[string]any `json:"action_args"`
	Result     any            `json:"result"`
	Success    bool           `json:"success"`
}

type TimelineEntry struct {
	RoundNum          int            `json:"round_num"`
	TwitterActions    int            `json:"twitter_actions"`
	RedditActions     int            `json:"reddit_actions"`
	TotalActions      int            `json:"total_actions"`
	ActiveAgentsCount int            `json:"active_agents_count"`
	ActiveAgents      []int          `json:"active_agents"`
	ActionTypes       map[string]int `json:"action_types"`
	FirstActionTime   string         `json:"first_action_time"`
	LastActionTime    string         `json:"last_action_time"`
}

type AgentStat struct {
	AgentID         int            `json:"agent_id"`
	AgentName       string         `json:"agent_name"`
	TotalActions    int            `json:"total_actions"`
	TwitterActions  int            `json:"twitter_actions"`
	RedditActions   int            `json:"reddit_actions"`
	ActionTypes     map[string]int `json:"action_types"`
	FirstActionTime string         `json:"first_action_time"`
	LastActionTime  string         `json:"last_action_time"`
}
