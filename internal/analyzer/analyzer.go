package analyzer

import (
	"context"
	"sort"

	"github.com/omerkarahan/loldmg/internal/models"
)

// Tunables configures fight detection thresholds.
type Tunables struct {
	KillClusterGap int // ms between kills to stay in one cluster
	PreKill        int // ms before first kill in cluster
	PostKill       int // ms after last kill in cluster
	MergeGap       int // ms between candidate windows to merge
}

// DefaultTunables returns sensible defaults.
func DefaultTunables() Tunables {
	return Tunables{
		KillClusterGap: 15000,
		PreKill:        10000,
		PostKill:       5000,
		MergeGap:       5000,
	}
}

// Analyzer runs offline fight detection on a match timeline.
type Analyzer struct {
	tunables Tunables
}

// New creates an Analyzer with default tunables.
func New() *Analyzer {
	return NewWithTunables(DefaultTunables())
}

// NewWithTunables creates an Analyzer with custom tunables.
func NewWithTunables(t Tunables) *Analyzer {
	return &Analyzer{tunables: t}
}

// DetectFights finds fight windows from a match timeline.
func (a *Analyzer) DetectFights(ctx context.Context, tl *models.Timeline) ([]models.Fight, error) {
	kills := collectKills(tl)
	if len(kills) == 0 {
		return nil, nil
	}
	objectives := collectObjectives(tl)

	clusters := clusterKills(kills, a.tunables.KillClusterGap)
	windows := buildWindows(clusters, a.tunables.PreKill, a.tunables.PostKill)
	windows = mergeWindows(windows, a.tunables.MergeGap)

	fights := make([]models.Fight, 0, len(windows))
	for _, w := range windows {
		fights = append(fights, a.buildFight(w, objectives))
	}
	return fights, nil
}

func collectKills(tl *models.Timeline) []models.Event {
	var kills []models.Event
	for _, frame := range tl.Info.Frames {
		for _, e := range frame.Events {
			if e.Type == "CHAMPION_KILL" {
				kills = append(kills, e)
			}
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].Timestamp < kills[j].Timestamp })
	return kills
}

func collectObjectives(tl *models.Timeline) []models.Event {
	var objs []models.Event
	for _, frame := range tl.Info.Frames {
		for _, e := range frame.Events {
			switch e.Type {
			case "ELITE_MONSTER_KILL", "BUILDING_KILL", "TURRET_PLATE_DESTROYED":
				objs = append(objs, e)
			}
		}
	}
	sort.Slice(objs, func(i, j int) bool { return objs[i].Timestamp < objs[j].Timestamp })
	return objs
}

func clusterKills(kills []models.Event, gap int) [][]models.Event {
	var clusters [][]models.Event
	for _, k := range kills {
		if len(clusters) == 0 {
			clusters = append(clusters, []models.Event{k})
			continue
		}
		last := clusters[len(clusters)-1]
		prev := last[len(last)-1]
		if k.Timestamp-prev.Timestamp <= gap {
			clusters[len(clusters)-1] = append(last, k)
		} else {
			clusters = append(clusters, []models.Event{k})
		}
	}
	return clusters
}

func buildWindows(clusters [][]models.Event, preKill, postKill int) []window {
	var windows []window
	for _, c := range clusters {
		minT, maxT := c[0].Timestamp, c[0].Timestamp
		for _, k := range c[1:] {
			if k.Timestamp < minT {
				minT = k.Timestamp
			}
			if k.Timestamp > maxT {
				maxT = k.Timestamp
			}
		}
		windows = append(windows, window{
			start: max(0, minT-preKill),
			end:   maxT + postKill,
			kills: c,
		})
	}
	return windows
}

func mergeWindows(windows []window, mergeGap int) []window {
	if len(windows) <= 1 {
		return windows
	}
	sort.Slice(windows, func(i, j int) bool { return windows[i].start < windows[j].start })
	merged := []window{windows[0]}
	for _, w := range windows[1:] {
		last := &merged[len(merged)-1]
		if w.start-last.end <= mergeGap {
			if w.end > last.end {
				last.end = w.end
			}
			last.kills = append(last.kills, w.kills...)
		} else {
			merged = append(merged, w)
		}
	}
	return merged
}

type window struct {
	start, end int
	kills      []models.Event
}

func (a *Analyzer) buildFight(w window, objectives []models.Event) models.Fight {
	dmgByParticipant := map[int]int{}
	killsByParticipant := map[int]int{}
	assistsByParticipant := map[int]int{}
	deathsByParticipant := map[int]int{}
	participantSet := map[int]struct{}{}

	for _, k := range w.kills {
		participantSet[k.KillerID] = struct{}{}
		killsByParticipant[k.KillerID]++
		for _, assistID := range k.AssistingParticipantIDs {
			participantSet[assistID] = struct{}{}
			assistsByParticipant[assistID]++
		}
		participantSet[k.VictimID] = struct{}{}
		deathsByParticipant[k.VictimID]++

		for _, dr := range k.VictimDamageReceived {
			if dr.ParticipantID <= 0 {
				continue
			}
			participantSet[dr.ParticipantID] = struct{}{}
			dmg := dr.MagicDamage + dr.PhysicalDamage + dr.TrueDamage
			dmgByParticipant[dr.ParticipantID] += dmg
		}
	}

	participants := make([]int, 0, len(participantSet))
	for p := range participantSet {
		participants = append(participants, p)
	}
	sort.Ints(participants)

	return models.Fight{
		StartMs:              w.start,
		EndMs:                w.end,
		DurationMs:           w.end - w.start,
		KillEvents:           w.kills,
		Participants:         participants,
		DamageByParticipant:  dmgByParticipant,
		KillsByParticipant:   killsByParticipant,
		AssistsByParticipant: assistsByParticipant,
		DeathsByParticipant:  deathsByParticipant,
		Label:                labelFight(w, objectives),
	}
}

func labelFight(w window, objectives []models.Event) string {
	for _, o := range objectives {
		if o.Timestamp < w.start || o.Timestamp > w.end {
			continue
		}
		switch o.Type {
		case "ELITE_MONSTER_KILL":
			switch o.MonsterType {
			case "BARON_NASHOR":
				return "Baron fight"
			case "ELDER_DRAGON", "DRAGON":
				return "Dragon fight"
			case "RIFT_HERALD":
				return "Herald fight"
			case "ATAKHAN":
				return "Atakhan fight"
			}
		case "BUILDING_KILL":
			switch o.BuildingType {
			case "INHIBITOR_BUILDING":
				return "Inhibitor fight"
			case "TOWER_BUILDING":
				return "Tower fight"
			}
		}
	}
	return "Skirmish"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
