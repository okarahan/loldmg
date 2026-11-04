package models

// Timeline is the top-level match-v5 timeline response.
type Timeline struct {
	Metadata TimelineMetadata `json:"metadata"`
	Info     TimelineInfo     `json:"info"`
}

// TimelineMetadata identifies the match.
type TimelineMetadata struct {
	DataVersion string   `json:"dataVersion"`
	MatchID     string   `json:"matchId"`
	Participants []string `json:"participants"`
}

// TimelineInfo holds frames and interval info.
type TimelineInfo struct {
	FrameInterval int     `json:"frameInterval"`
	Frames        []Frame `json:"frames"`
}

// Frame is one minute-ish snapshot with events and participant frames.
type Frame struct {
	Events            []Event                       `json:"events"`
	ParticipantFrames map[string]ParticipantFrame   `json:"participantFrames"`
	Timestamp         int                           `json:"timestamp"`
}

// Event represents a timeline event.
type Event struct {
	Type                    string           `json:"type"`
	Timestamp               int              `json:"timestamp"`
	KillerID                int              `json:"killerId,omitempty"`
	VictimID                int              `json:"victimId,omitempty"`
	AssistingParticipantIDs []int            `json:"assistingParticipantIds,omitempty"`
	KillStreakLength        int              `json:"killStreakLength,omitempty"`
	VictimDamageReceived    []DamageReceived `json:"victimDamageReceived,omitempty"`
	VictimDamageDealt       []DamageReceived `json:"victimDamageDealt,omitempty"`
	// Objective events.
	MonsterType  string `json:"monsterType,omitempty"`
	BuildingType string `json:"buildingType,omitempty"`
	TowerType    string `json:"towerType,omitempty"`
	LaneType     string `json:"laneType,omitempty"`
	TeamID       int    `json:"teamId,omitempty"`
}

// DamageReceived describes damage from a specific source.
type DamageReceived struct {
	ParticipantID  int    `json:"participantId"`
	Name         string `json:"name"`
	MagicDamage  int    `json:"magicDamage"`
	PhysicalDamage int  `json:"physicalDamage"`
	TrueDamage   int    `json:"trueDamage"`
	SpellName    string `json:"spellName,omitempty"`
	SpellSlot    int    `json:"spellSlot,omitempty"`
	Basic        bool   `json:"basic"`
}

// ParticipantFrame is a per-minute snapshot of a participant.
type ParticipantFrame struct {
	ParticipantID   int            `json:"participantId"`
	ChampionStats   ChampionStats  `json:"championStats"`
	DamageStats     DamageStats    `json:"damageStats"`
	Level           int            `json:"level"`
}

// ChampionStats holds live stats at a frame timestamp.
type ChampionStats struct {
	Health    int `json:"health"`
	HealthMax int `json:"healthMax"`
}

// DamageStats holds cumulative damage at a frame timestamp.
type DamageStats struct {
	MagicDamageDoneToChampions    int `json:"magicDamageDoneToChampions"`
	PhysicalDamageDoneToChampions int `json:"physicalDamageDoneToChampions"`
	TrueDamageDoneToChampions     int `json:"trueDamageDoneToChampions"`
	TotalDamageDoneToChampions    int `json:"totalDamageDoneToChampions"`
}

// ParticipantInfo maps participantId to identity (from match info endpoint).
type ParticipantInfo struct {
	ParticipantID  int    `json:"participantId"`
	PUUID          string `json:"puuid"`
	SummonerName   string `json:"summonerName"`
	RiotIDGameName string `json:"riotIdGameName"`
	RiotIDTagline  string `json:"riotIdTagline"`
	ChampionName   string `json:"championName"`
	TeamID         int    `json:"teamId"`
}

// MatchInfo is the response from /lol/match/v5/matches/{matchId}.
type MatchInfo struct {
	Info struct {
		Participants []ParticipantInfo `json:"participants"`
	} `json:"info"`
}

// Fight is the result of analyzing the timeline.
type Fight struct {
	StartMs               int                       `json:"start_ms"`
	EndMs                 int                       `json:"end_ms"`
	DurationMs            int                       `json:"duration_ms"`
	KillEvents            []Event                   `json:"kill_events"`
	Participants          []int                     `json:"participants"`
	DamageByParticipant   map[int]int               `json:"damage_by_participant"`
	KillsByParticipant    map[int]int               `json:"kills_by_participant"`
	AssistsByParticipant  map[int]int               `json:"assists_by_participant"`
	DeathsByParticipant   map[int]int               `json:"deaths_by_participant"`
	Label                 string                    `json:"label"`
}
