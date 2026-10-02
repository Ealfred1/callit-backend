package mailer

import (
	"strings"
	"testing"
)

func TestRenderTemplates(t *testing.T) {
	t.Run("login template", func(t *testing.T) {
		html, err := renderTemplate("login.html", map[string]string{
			"verification_code": "849201",
			"support_email":     "support@callit.my",
		})
		if err != nil {
			t.Fatalf("unexpected error rendering login.html: %v", err)
		}
		if !strings.Contains(html, "849201") {
			t.Errorf("expected rendered html to contain code 849201")
		}
		if !strings.Contains(html, "support@callit.my") {
			t.Errorf("expected rendered html to contain support email")
		}
		if strings.Contains(html, "{{") || strings.Contains(html, "}}") {
			t.Errorf("expected no unreplaced placeholders in login.html")
		}
	})

	t.Run("welcome template", func(t *testing.T) {
		html, err := renderTemplate("welcome.html", map[string]string{
			"first_name":       "Eric",
			"app_link":         "https://callit.my",
			"support_email":    "support@callit.my",
			"unsubscribe_link": "https://callit.my/unsubscribe",
		})
		if err != nil {
			t.Fatalf("unexpected error rendering welcome.html: %v", err)
		}
		if !strings.Contains(html, "Welcome to Callit, Eric.") {
			t.Errorf("expected rendered html to contain welcome greeting")
		}
		if !strings.Contains(html, "https://callit.my") {
			t.Errorf("expected rendered html to contain app link")
		}
		if strings.Contains(html, "{{") || strings.Contains(html, "}}") {
			t.Errorf("expected no unreplaced placeholders in welcome.html")
		}
	})

	t.Run("layout template", func(t *testing.T) {
		html, err := renderTemplate("layout.html", map[string]string{
			"preheader":        "Your market resolved",
			"heading":          "Market Resolved: YES",
			"body":             "You have 45.00 USDC in winnings ready to claim.",
			"button_label":     "Claim Winnings",
			"button_url":       "https://callit.my/claim",
			"app_link":         "https://callit.my/claim",
			"support_email":    "support@callit.my",
			"unsubscribe_link": "https://callit.my/unsubscribe",
		})
		if err != nil {
			t.Fatalf("unexpected error rendering layout.html: %v", err)
		}
		if !strings.Contains(html, "Market Resolved: YES") {
			t.Errorf("expected rendered html to contain heading")
		}
		if !strings.Contains(html, "Claim Winnings") {
			t.Errorf("expected rendered html to contain button label")
		}
		if strings.Contains(html, "{{") || strings.Contains(html, "}}") {
			t.Errorf("expected no unreplaced placeholders in layout.html")
		}
	})
}
