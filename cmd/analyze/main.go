package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"time"

	"github.com/omerkarahan/loldmg/internal/analyzer"
	"github.com/omerkarahan/loldmg/internal/matchv5"
	"github.com/omerkarahan/loldmg/internal/models"
)

func main() {
	var (
		apiKey       = flag.String("api-key", os.Getenv("RIOT_API_KEY"), "Riot API key (defaults to RIOT_API_KEY env var)")
		region       = flag.String("region", "europe", "Riot routing region: americas, europe, asia, sea, esports")
		matchID      = flag.String("match-id", "", "match ID (required unless --timeline-file is set)")
		timelineFile = flag.String("timeline-file", "", "local timeline JSON file (for testing)")
		matchFile    = flag.String("match-file", "", "local match info JSON file (for testing)")
		format       = flag.String("format", "text", "output format: text or json")
		killClusterGap = flag.Int("kill-cluster-gap", 15000, "ms between kills to stay in one cluster")
		preKill      = flag.Int("pre-kill", 10000, "ms before first kill in cluster")
		postKill     = flag.Int("post-kill", 5000, "ms after last kill in cluster")
		mergeGap     = flag.Int("merge-gap", 5000, "ms between candidate windows to merge")
	)
	flag.Parse()

	var tl *models.Timeline
	var info *models.MatchInfo
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if *timelineFile != "" {
		tl, err = loadTimeline(*timelineFile)
		if err != nil {
			log.Fatalf("load timeline: %v", err)
		}
		if *matchFile != "" {
			info, err = loadMatchInfo(*matchFile)
			if err != nil {
				log.Fatalf("load match info: %v", err)
			}
		}
	} else {
		if *apiKey == "" || *matchID == "" {
			flag.Usage()
			os.Exit(1)
		}
		client := matchv5.New(*apiKey, matchv5.WithRegion(*region))
		tl, err = client.GetTimeline(ctx, *matchID)
		if err != nil {
			log.Fatalf("get timeline: %v", err)
		}
		info, err = client.GetMatchInfo(ctx, *matchID)
		if err != nil {
			log.Fatalf("get match info: %v", err)
		}
	}

	t := analyzer.Tunables{
		KillClusterGap: *killClusterGap,
		PreKill:        *preKill,
		PostKill:       *postKill,
		MergeGap:       *mergeGap,
	}

	fights, err := analyzer.NewWithTunables(t).DetectFights(ctx, tl)
	if err != nil {
		log.Fatalf("detect fights: %v", err)
	}

	identities := buildIdentities(info)
	if hasUnresolvedParticipants(fights, identities) {
		fmt.Fprintln(os.Stderr, "warning: some participants could not be resolved to summoner names.")
		if *timelineFile != "" {
			fmt.Fprintln(os.Stderr, "         provide --match-file with the match info JSON, or use --api-key and --match-id.")
		} else {
			fmt.Fprintln(os.Stderr, "         the match info response may be missing participant names.")
		}
	}
	out := formatFights(fights, identities)

	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			log.Fatalf("encode: %v", err)
		}
	case "text":
		printTextFights(out, os.Stdout)
	default:
		log.Fatalf("unknown format: %s", *format)
	}
}

func hasUnresolvedParticipants(fights []models.Fight, identities map[int]identity) bool {
	for _, f := range fights {
		for _, pid := range f.Participants {
			if _, ok := identities[pid]; !ok {
				return true
			}
		}
	}
	return false
}

func loadTimeline(path string) (*models.Timeline, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var tl models.Timeline
	return &tl, json.NewDecoder(f).Decode(&tl)
}

func loadMatchInfo(path string) (*models.MatchInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var mi models.MatchInfo
	return &mi, json.NewDecoder(f).Decode(&mi)
}

func buildIdentities(info *models.MatchInfo) map[int]identity {
	m := map[int]identity{}
	if info == nil {
		return m
	}
	for _, p := range info.Info.Participants {
		m[p.ParticipantID] = identity{
			DisplayName:  displayName(p),
			ChampionName: p.ChampionName,
		}
	}
	return m
}

func displayName(p models.ParticipantInfo) string {
	if p.RiotIDGameName != "" {
		if p.RiotIDTagline != "" {
			return p.RiotIDGameName + "#" + p.RiotIDTagline
		}
		return p.RiotIDGameName
	}
	if p.SummonerName != "" {
		return p.SummonerName
	}
	return ""
}

type identity struct {
	DisplayName  string `json:"display_name"`
	ChampionName string `json:"champion_name"`
}

func formatFights(fights []models.Fight, identities map[int]identity) []formattedFight {
	out := make([]formattedFight, 0, len(fights))
	for _, f := range fights {
		dmg := map[string]int{}
		for pid, d := range f.DamageByParticipant {
			dmg[key(pid, identities)] = d
		}
		kills := map[string]int{}
		for pid, k := range f.KillsByParticipant {
			kills[key(pid, identities)] = k
		}
		assists := map[string]int{}
		for pid, a := range f.AssistsByParticipant {
			assists[key(pid, identities)] = a
		}
		deaths := map[string]int{}
		for pid, d := range f.DeathsByParticipant {
			deaths[key(pid, identities)] = d
		}
		out = append(out, formattedFight{
			StartMs:         f.StartMs,
			EndMs:           f.EndMs,
			DurationMs:      f.DurationMs,
			Label:           f.Label,
			Kills:           len(f.KillEvents),
			Participants:    formatParticipants(f.Participants, identities),
			Damage:          dmg,
			KillsByPlayer:   kills,
			AssistsByPlayer: assists,
			DeathsByPlayer:  deaths,
		})
	}
	return out
}

func key(pid int, identities map[int]identity) string {
	if id, ok := identities[pid]; ok && id.ChampionName != "" {
		if id.DisplayName != "" {
			return fmt.Sprintf("%s (%s)", id.ChampionName, id.DisplayName)
		}
		return id.ChampionName
	}
	return fmt.Sprintf("participant_%d", pid)
}

func formatParticipants(pids []int, identities map[int]identity) []string {
	out := make([]string, 0, len(pids))
	for _, pid := range pids {
		out = append(out, key(pid, identities))
	}
	sortStrings(out)
	return out
}

func sortStrings(s []string) {
	sort.Strings(s)
}

type formattedFight struct {
	StartMs         int            `json:"start_ms"`
	EndMs           int            `json:"end_ms"`
	DurationMs      int            `json:"duration_ms"`
	Label           string         `json:"label"`
	Kills           int            `json:"kills"`
	Participants    []string       `json:"participants"`
	Damage          map[string]int `json:"damage"`
	KillsByPlayer   map[string]int `json:"kills_by_player"`
	AssistsByPlayer map[string]int `json:"assists_by_player"`
	DeathsByPlayer  map[string]int `json:"deaths_by_player"`
}

func printTextFights(fights []formattedFight, w io.Writer) {
	for i, f := range fights {
		startMin := f.StartMs / 60000
		startSec := (f.StartMs % 60000) / 1000
		endMin := f.EndMs / 60000
		endSec := (f.EndMs % 60000) / 1000
		durSec := f.DurationMs / 1000

		fmt.Fprintf(w, "\nFight %d  @ %02d:%02d - %02d:%02d (%ds) | %s | %d kill(s)\n", i+1, startMin, startSec, endMin, endSec, durSec, f.Label, f.Kills)

		players := make([]string, 0, len(f.Participants))
		for _, p := range f.Participants {
			players = append(players, p)
		}
		sortStrings(players)

		maxDmg := 0
		for _, p := range players {
			if d := f.Damage[p]; d > maxDmg {
				maxDmg = d
			}
		}
		if maxDmg == 0 {
			maxDmg = 1
		}

		for _, p := range players {
			dmg := f.Damage[p]
			bar := ""
			if dmg > 0 {
				bar = damageBar(dmg, maxDmg, 20)
			}
			fmt.Fprintf(w, "  %-28s %5d  %s\n", p, dmg, bar)
		}
		fmt.Fprintln(w)
	}
	if len(fights) == 0 {
		fmt.Fprintln(w, "No fights detected.")
	}
}

func damageBar(value, max, width int) string {
	if max <= 0 {
		return ""
	}
	filled := value * width / max
	if filled > width {
		filled = width
	}
	if filled <= 0 {
		return ""
	}
	b := make([]rune, width)
	for i := range width {
		if i < filled {
			b[i] = '█'
		} else {
			b[i] = '░'
		}
	}
	return string(b)
}
