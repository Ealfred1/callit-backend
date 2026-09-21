package service

import (
	"context"
	"os"
	"path/filepath"

	"github.com/eric/callit-backend/internal/adapter/handler/respond"
	"github.com/eric/callit-backend/internal/adapter/sharecard"
	"github.com/eric/callit-backend/internal/core/ports"
)

type ShareCardService struct {
	panta      ports.PantaClient
	marketMeta ports.MarketMetaRepository
	staticDir  string
	publicURL  string
}

func NewShareCardService(panta ports.PantaClient, marketMeta ports.MarketMetaRepository, staticDir, publicURL string) *ShareCardService {
	return &ShareCardService{panta: panta, marketMeta: marketMeta, staticDir: staticDir, publicURL: publicURL}
}

type CardResult struct {
	ImageURL string
	Title    string
	Category string
}

// RenderAndStore fetches live market data, draws the card, and writes it to
// ./static/share-cards/<marketId>.png (served at /static/share-cards/...).
func (s *ShareCardService) RenderAndStore(ctx context.Context, marketID string) (*CardResult, error) {
	market, err := s.panta.GetMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}

	yes, no := 0.5, 0.5
	if market.YesPrice != nil {
		yes = parseFloatSafe(*market.YesPrice)
	} else if market.PrimaryYesPrice != nil {
		yes = parseFloatSafe(*market.PrimaryYesPrice)
	}
	if market.NoPrice != nil {
		no = parseFloatSafe(*market.NoPrice)
	} else if market.PrimaryNoPrice != nil {
		no = parseFloatSafe(*market.PrimaryNoPrice)
	}

	png, err := sharecard.Render(sharecard.CardInput{Question: market.Title, Category: string(market.Category), YesPrice: yes, NoPrice: no})
	if err != nil {
		return nil, respond.NewAppError(500, "SHARE_CARD_RENDER_FAILED", err.Error())
	}

	dir := filepath.Join(s.staticDir, "share-cards")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, marketID+".png")
	if err := os.WriteFile(path, png, 0o644); err != nil {
		return nil, err
	}

	imageURL := s.publicURL + "/static/share-cards/" + marketID + ".png"
	_ = s.marketMeta.SetShareImagePath(ctx, marketID, "/static/share-cards/"+marketID+".png")

	return &CardResult{ImageURL: imageURL, Title: market.Title, Category: string(market.Category)}, nil
}

// GetOrRender reuses a cached card if market_meta already has one, else
// renders on the fly — so a share link never breaks just because the
// create flow didn't pre-warm it.
func (s *ShareCardService) GetOrRender(ctx context.Context, marketID string) (*CardResult, error) {
	if meta, err := s.marketMeta.GetByMarketID(ctx, marketID); err == nil && meta.ShareImagePath != nil && *meta.ShareImagePath != "" {
		return &CardResult{ImageURL: s.publicURL + *meta.ShareImagePath, Title: meta.Title, Category: meta.Category}, nil
	}
	return s.RenderAndStore(ctx, marketID)
}
