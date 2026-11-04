package analyzer_test

import (
	"context"
	"testing"

	"github.com/omerkarahan/loldmg/internal/analyzer"
	"github.com/omerkarahan/loldmg/internal/models"
)

func makeTimeline() *models.Timeline {
	return &models.Timeline{
		Metadata: models.TimelineMetadata{MatchID: "TEST_1"},
		Info: models.TimelineInfo{
			Frames: []models.Frame{
				{
					Timestamp: 0,
					Events: []models.Event{
						{
							Type:                    "CHAMPION_KILL",
							Timestamp:               60000,
							KillerID:                1,
							VictimID:                5,
							AssistingParticipantIDs: []int{2},
							VictimDamageReceived: []models.DamageReceived{
								{ParticipantID: 1, MagicDamage: 200, PhysicalDamage: 300, TrueDamage: 0},
								{ParticipantID: 2, MagicDamage: 100, PhysicalDamage: 0, TrueDamage: 50},
							},
						},
						{
							Type:                    "CHAMPION_KILL",
							Timestamp:               65000,
							KillerID:                1,
							VictimID:                6,
							AssistingParticipantIDs: []int{3},
							VictimDamageReceived: []models.DamageReceived{
								{ParticipantID: 1, MagicDamage: 0, PhysicalDamage: 400, TrueDamage: 100},
								{ParticipantID: 3, MagicDamage: 150, PhysicalDamage: 0, TrueDamage: 0},
							},
						},
					},
				},
				{
					Timestamp: 60000,
					Events: []models.Event{
						{
							Type:      "CHAMPION_KILL",
							Timestamp: 180000,
							KillerID:  5,
							VictimID:  1,
							VictimDamageReceived: []models.DamageReceived{
								{ParticipantID: 5, MagicDamage: 0, PhysicalDamage: 250, TrueDamage: 0},
							},
						},
					},
				},
			},
		},
	}
}

func TestDetectFightsTwoKills(t *testing.T) {
	tl := makeTimeline()
	a := analyzer.New()
	fights, err := a.DetectFights(context.Background(), tl)
	if err != nil {
		t.Fatalf("DetectFights: %v", err)
	}
	if len(fights) != 2 {
		t.Fatalf("expected 2 fights, got %d", len(fights))
	}

	// First fight: two kills within 5 seconds.
	f := fights[0]
	if len(f.KillEvents) != 2 {
		t.Errorf("expected 2 kills in first fight, got %d", len(f.KillEvents))
	}
	if f.DamageByParticipant[1] != 1000 {
		t.Errorf("participant 1 damage = %v, want 1000", f.DamageByParticipant[1])
	}
	if f.DamageByParticipant[2] != 150 {
		t.Errorf("participant 2 damage = %v, want 150", f.DamageByParticipant[2])
	}
	if f.DamageByParticipant[3] != 150 {
		t.Errorf("participant 3 damage = %v, want 150", f.DamageByParticipant[3])
	}
	if f.KillsByParticipant[1] != 2 {
		t.Errorf("participant 1 kills = %v, want 2", f.KillsByParticipant[1])
	}
	if f.AssistsByParticipant[2] != 1 {
		t.Errorf("participant 2 assists = %v, want 1", f.AssistsByParticipant[2])
	}
	if f.DeathsByParticipant[5] != 1 {
		t.Errorf("participant 5 deaths = %v, want 1", f.DeathsByParticipant[5])
	}
}

func TestDetectFightsNoKills(t *testing.T) {
	tl := &models.Timeline{}
	a := analyzer.New()
	fights, err := a.DetectFights(context.Background(), tl)
	if err != nil {
		t.Fatalf("DetectFights: %v", err)
	}
	if len(fights) != 0 {
		t.Fatalf("expected 0 fights, got %d", len(fights))
	}
}
