package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/omerkarahan/loldmg/internal/analyzer"
	"github.com/omerkarahan/loldmg/internal/matchv5"
	"github.com/omerkarahan/loldmg/internal/models"
)

//go:embed static
var staticFS embed.FS

// Server is the HTTP frontend.
type Server struct {
	apiKey string
	region string
	devPath string
}

// New creates a web server with default Riot API config.
func New(apiKey, region, devPath string) *Server {
	if apiKey == "" {
		apiKey = os.Getenv("RIOT_API_KEY")
	}
	if region == "" {
		region = "europe"
	}
	return &Server{apiKey: apiKey, region: region, devPath: devPath}
}

// Handler returns the HTTP mux.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.HandleFunc("/api/analyze", s.handleAnalyze)
	return mux
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if s.devPath != "" {
		s.handleDevAnalyze(w, r)
		return
	}
	matchID := r.URL.Query().Get("match_id")
	region := r.URL.Query().Get("region")
	if region == "" {
		region = s.region
	}
	if matchID == "" {
		http.Error(w, `{"error":"match_id required"}`, http.StatusBadRequest)
		return
	}
	if s.apiKey == "" {
		http.Error(w, `{"error":"RIOT_API_KEY not configured"}`, http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	client := matchv5.New(s.apiKey, matchv5.WithRegion(region))
	tl, err := client.GetTimeline(ctx, matchID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"timeline: %s"}`, err.Error()), http.StatusBadGateway)
		return
	}
	info, err := client.GetMatchInfo(ctx, matchID)
	if err != nil {
		// Non-fatal: we can still analyze without names.
		info = nil
	}

	fights, err := analyzer.New().DetectFights(ctx, tl)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"analyze: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	resp := analyzeResponse{
		MatchID: matchID,
		Region:  region,
		Fights:  fights,
		Info:    info,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleDevAnalyze(w http.ResponseWriter, r *http.Request) {
	tlPath := s.devPath + "/timeline.json"
	matchPath := s.devPath + "/match.json"
	tlFile, err := os.ReadFile(tlPath)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"timeline file: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	matchFile, err := os.ReadFile(matchPath)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"match file: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	var tl models.Timeline
	if err := json.Unmarshal(tlFile, &tl); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"parse timeline: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	var info models.MatchInfo
	_ = json.Unmarshal(matchFile, &info)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	fights, err := analyzer.New().DetectFights(ctx, &tl)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"analyze: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(analyzeResponse{
		MatchID: "dev-match",
		Region:  s.region,
		Fights:  fights,
		Info:    &info,
	})
}

type analyzeResponse struct {
	MatchID string            `json:"match_id"`
	Region  string            `json:"region"`
	Fights  []models.Fight    `json:"fights"`
	Info    *models.MatchInfo `json:"info,omitempty"`
}
