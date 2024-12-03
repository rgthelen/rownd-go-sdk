package rownd_test

import (
	"context"
	"strings"
	"testing"

	"github.com/rgthelen/rownd-go-sdk/internal/testutils"
	"github.com/rgthelen/rownd-go-sdk/pkg/rownd"
	"github.com/stretchr/testify/assert"
)

func TestRowndToken(t *testing.T) {
	// Get test configuration
	testConfig := testutils.GetTestConfig()
	var validToken string // Will be set after magic link redemption

	client, err := rownd.NewClient(
		rownd.WithAppKey(testConfig.AppKey),
		rownd.WithAppSecret(testConfig.AppSecret),
		rownd.WithAppID(testConfig.AppID),
		rownd.WithBaseURL(testConfig.BaseURL),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Start with smart links tests to get our token
	t.Run("magic links", func(t *testing.T) {
		var magicLinkUserID string

		t.Run("create magic link", func(t *testing.T) {
			opts := &rownd.MagicLinkOptions{
				Purpose:          "auth",
				VerificationType: "email",
				Data: map[string]interface{}{
					"email":      "testlink@example.com",
					"first_name": "Test",
				},
				RedirectURL: "https://example.com/redirect",
				Expiration:  "30d",
			}

			req := rownd.CreateMagicLinkRequest{
				Purpose:          rownd.Purpose(opts.Purpose),
				VerificationType: rownd.VerificationType(opts.VerificationType),
				Data:             opts.Data,
				RedirectURL:      opts.RedirectURL,
				Expiration:       opts.Expiration,
			}

			link, err := client.MagicLinks.Create(ctx, req)
			assert.NoError(t, err)
			assert.NotNil(t, link)
			assert.NotEmpty(t, link.Link)

			// Extract link ID and redeem it
			parts := strings.Split(link.Link, "/")
			linkID := parts[len(parts)-1]

			magicLinkResp, err := testutils.RedeemMagicLink(ctx, client, linkID)
			assert.NoError(t, err)
			assert.NotNil(t, magicLinkResp)

			// Store the token and user ID for subsequent tests and cleanup
			validToken = magicLinkResp.AccessToken
			magicLinkUserID = magicLinkResp.AppUserID

			t.Logf("Created and redeemed magic link for user: %s", magicLinkUserID)
		})

		// Add cleanup at the test suite level to ensure it runs after all tests
		t.Cleanup(func() {
			if magicLinkUserID != "" {
				err := client.Users.Delete(ctx, rownd.DeleteUserRequest{
					AppID:  testConfig.AppID,
					UserID: magicLinkUserID,
				})
				if err != nil {
					t.Logf("Failed to cleanup magic link user %s: %v", magicLinkUserID, err)
				} else {
					t.Logf("Cleaned up magic link user: %s", magicLinkUserID)
				}
			}
		})
	})

	// Token validation tests using magic link token
	t.Run("token validation", func(t *testing.T) {
		if validToken == "" {
			t.Fatal("No valid token available from magic link")
		}

		t.Run("validate token", func(t *testing.T) {
			tokenInfo, err := testutils.ValidateTokenForTest(ctx, client, validToken)
			if err != nil {
				t.Fatalf("Failed to validate token: %v", err)
			}
			assert.NotNil(t, tokenInfo)
			assert.NotEmpty(t, tokenInfo.UserID)
		})

		t.Run("extract token claims", func(t *testing.T) {
			tokenInfo, err := testutils.ValidateTokenForTest(ctx, client, validToken)
			assert.NoError(t, err)
			assert.NotNil(t, tokenInfo)

			// Check claims
			userID := tokenInfo.Claims.AppUserID
			assert.NotEmpty(t, userID)
			t.Logf("User ID from token: %s", userID)

			isVerified := tokenInfo.Claims.IsUserVerified
			t.Logf("User verified status: %v", isVerified)

			authLevel := tokenInfo.Claims.AuthLevel
			assert.NotEmpty(t, authLevel)
			t.Logf("Auth level: %s", authLevel)
		})
	})
}
