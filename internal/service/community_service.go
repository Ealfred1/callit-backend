package service

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/core/domain"
	"github.com/eric/callit-backend/internal/core/ports"
	"github.com/google/uuid"
)

type CommunityService struct {
	communities ports.CommunityRepository
	marketMeta  ports.MarketMetaRepository
	panta       ports.PantaClient
}

func NewCommunityService(communities ports.CommunityRepository, marketMeta ports.MarketMetaRepository, panta ports.PantaClient) *CommunityService {
	return &CommunityService{communities: communities, marketMeta: marketMeta, panta: panta}
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	s := slugRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	s = strings.Trim(s, "-")
	if len(s) > 48 {
		s = s[:48]
	}
	if s == "" {
		s = "community"
	}
	return s
}

func (s *CommunityService) Create(ctx context.Context, creatorUserID, name string) (*domain.Community, error) {
	if len(strings.TrimSpace(name)) < 2 {
		return nil, respond.BadRequest("INVALID_MARKET_PARAMS", "name is required")
	}
	creatorUUID, err := uuid.Parse(creatorUserID)
	if err != nil {
		return nil, respond.Unauthorized("invalid user")
	}

	base := slugify(name)
	slug := base
	// small retry loop for the rare slug collision, rather than a
	// pre-check + insert race.
	for attempt := 0; attempt < 6; attempt++ {
		c := &domain.Community{
			Name: strings.TrimSpace(name), Slug: slug, AttributionID: "cty_" + slug, CreatorUserID: creatorUUID,
		}
		err := s.communities.Create(ctx, c)
		if err == nil {
			return c, nil
		}
		if !strings.Contains(err.Error(), "duplicate key") {
			return nil, err
		}
		slug = base + "-" + strconv.Itoa(attempt+1)
	}
	return nil, respond.NewAppError(500, "DB_ERROR", "could not allocate a unique community slug")
}

type CommunityWithVolume struct {
	domain.Community
	MarketCount int     `json:"marketCount"`
	VolumeUsdc  float64 `json:"volumeUsdc"`
}

func (s *CommunityService) volumeFor(ctx context.Context, communityID string) (int, float64) {
	markets, err := s.marketMeta.ListByCommunity(ctx, communityID)
	if err != nil {
		return 0, 0
	}
	var total float64
	for _, m := range markets {
		if live, err := s.panta.GetMarket(ctx, m.MarketID); err == nil {
			total += parseFloatSafe(live.VolumeUsdc)
		}
	}
	return len(markets), total
}

func (s *CommunityService) List(ctx context.Context) ([]CommunityWithVolume, error) {
	communities, err := s.communities.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CommunityWithVolume, 0, len(communities))
	for _, c := range communities {
		count, vol := s.volumeFor(ctx, c.ID.String())
		out = append(out, CommunityWithVolume{Community: c, MarketCount: count, VolumeUsdc: vol})
	}
	return out, nil
}

func (s *CommunityService) GetByID(ctx context.Context, id string) (*CommunityWithVolume, []domain.MarketMeta, error) {
	c, err := s.communities.GetByID(ctx, id)
	if err != nil {
		return nil, nil, respond.NotFound("COMMUNITY_NOT_FOUND", "no community with that id")
	}
	markets, _ := s.marketMeta.ListByCommunity(ctx, id)
	var total float64
	for _, m := range markets {
		if live, err := s.panta.GetMarket(ctx, m.MarketID); err == nil {
			total += parseFloatSafe(live.VolumeUsdc)
		}
	}
	return &CommunityWithVolume{Community: *c, MarketCount: len(markets), VolumeUsdc: total}, markets, nil
}

// ResolveAttributionID turns an (optional) community id into the
// attribution_id that's sent to Panta as the buy-flow userId — resolved
// server-side so a client can't spoof arbitrary attribution strings.
func (s *CommunityService) ResolveAttributionID(ctx context.Context, communityID string) string {
	if communityID == "" {
		return ""
	}
	c, err := s.communities.GetByID(ctx, communityID)
	if err != nil {
		return ""
	}
	return c.AttributionID
}

func parseFloatSafe(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
