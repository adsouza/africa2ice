package render

import "testing"

func TestNewCampaignButtonContainsUsesHalfOpenBounds(t *testing.T) {
	if !NewCampaignButtonContains(newCampaignButtonX, newCampaignButtonY) ||
		!NewCampaignButtonContains(newCampaignButtonX+newCampaignButtonWidth-1, newCampaignButtonY+newCampaignButtonHeight-1) {
		t.Fatal("button rejected an interior point")
	}
	if NewCampaignButtonContains(newCampaignButtonX-1, newCampaignButtonY) ||
		NewCampaignButtonContains(newCampaignButtonX+newCampaignButtonWidth, newCampaignButtonY) {
		t.Fatal("button accepted an exterior point")
	}
}
