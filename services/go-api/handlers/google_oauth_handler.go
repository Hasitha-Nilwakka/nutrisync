// This file implements HTTP handlers for Google OAuth 2.0 login initiation (GET /api/auth/oauth/google)
// and callback handling with federated account linking (GET /api/auth/oauth/google/callback).
package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"nutrysync/go-api/auth"
	"nutrysync/go-api/models"
	"nutrysync/go-api/repositories"
)

const (
	googleOAuthProviderName       = "google"
	defaultGoogleUserInfoEndpoint = "https://openidconnect.googleapis.com/v1/userinfo"
	defaultFrontendAppBaseURL     = "http://localhost:3000"
)

// GoogleOpenIDUserInfoPayload represents the JSON payload returned by Google's OpenID Connect userinfo endpoint.
type GoogleOpenIDUserInfoPayload struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// GoogleOAuthHandler coordinates Google OAuth 2.0 login redirects, callback verification,
// federated account linking, and session issuance.
type GoogleOAuthHandler struct {
	accountRepository   repositories.AccountRepository
	sessionRepository   repositories.UserSessionRepository
	oauthConfig         *oauth2.Config
	jwtSecret           string
	appBaseURL          string
	isProduction        bool
	userInfoEndpointURL string
	httpClient          *http.Client
}

// NewGoogleOAuthHandler constructs a GoogleOAuthHandler after validating required dependencies.
func NewGoogleOAuthHandler(
	accountRepository repositories.AccountRepository,
	sessionRepository repositories.UserSessionRepository,
	oauthConfig *oauth2.Config,
	jwtSecret string,
	appBaseURL string,
	isProduction bool,
) (*GoogleOAuthHandler, error) {
	if accountRepository == nil {
		return nil, errors.New("account repository must not be nil")
	}

	if sessionRepository == nil {
		return nil, errors.New("session repository must not be nil")
	}

	if oauthConfig == nil {
		return nil, errors.New("oauth config must not be nil")
	}

	trimmedJWTSecret := strings.TrimSpace(jwtSecret)
	if trimmedJWTSecret == "" {
		return nil, errors.New("JWT secret must not be empty")
	}

	normalizedBaseURL := strings.TrimRight(strings.TrimSpace(appBaseURL), "/")
	if normalizedBaseURL == "" {
		normalizedBaseURL = defaultFrontendAppBaseURL
	}

	return &GoogleOAuthHandler{
		accountRepository:   accountRepository,
		sessionRepository:   sessionRepository,
		oauthConfig:         oauthConfig,
		jwtSecret:           trimmedJWTSecret,
		appBaseURL:          normalizedBaseURL,
		isProduction:        isProduction,
		userInfoEndpointURL: defaultGoogleUserInfoEndpoint,
		httpClient:          http.DefaultClient,
	}, nil
}

// HandleGoogleOAuthLogin handles GET /api/auth/oauth/google:
// generates a random CSRF state token, stores it in an HttpOnly cookie, and redirects (307) to Google's consent screen.
func (handler *GoogleOAuthHandler) HandleGoogleOAuthLogin(
	responseWriter http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodGet {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only GET method is supported",
		})
		return
	}

	stateToken, stateError := auth.GenerateRandomOAuthState()
	if stateError != nil {
		http.Redirect(responseWriter, request, handler.appBaseURL+"/?error=oauth_failed", http.StatusSeeOther)
		return
	}

	auth.SetOAuthStateCookie(responseWriter, stateToken, handler.isProduction)
	consentRedirectURL := handler.oauthConfig.AuthCodeURL(stateToken, oauth2.AccessTypeOffline)
	http.Redirect(responseWriter, request, consentRedirectURL, http.StatusTemporaryRedirect)
}

// fetchGoogleUserInfo retrieves and parses the authenticated user's OpenID Connect profile using the OAuth access token.
func (handler *GoogleOAuthHandler) fetchGoogleUserInfo(
	ctx context.Context,
	accessToken string,
) (*GoogleOpenIDUserInfoPayload, error) {
	endpointURL := strings.TrimSpace(handler.userInfoEndpointURL)
	if endpointURL == "" {
		endpointURL = defaultGoogleUserInfoEndpoint
	}

	userInfoRequest, requestError := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
	if requestError != nil {
		return nil, requestError
	}
	userInfoRequest.Header.Set("Authorization", "Bearer "+accessToken)

	httpClient := handler.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	response, responseError := httpClient.Do(userInfoRequest)
	if responseError != nil {
		return nil, responseError
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned status %d", response.StatusCode)
	}

	var userInfo GoogleOpenIDUserInfoPayload
	if decodeError := json.NewDecoder(response.Body).Decode(&userInfo); decodeError != nil {
		return nil, decodeError
	}

	if strings.TrimSpace(userInfo.Sub) == "" || strings.TrimSpace(userInfo.Email) == "" || !userInfo.EmailVerified {
		return nil, errors.New("invalid or unverified Google user profile")
	}

	return &userInfo, nil
}

// resolveOrCreateFederatedUser resolves an existing User by Account or Email, or creates a new OAuth User + Account.
func (handler *GoogleOAuthHandler) resolveOrCreateFederatedUser(
	ctx context.Context,
	userInfo *GoogleOpenIDUserInfoPayload,
) (*models.User, error) {
	existingAccount, accountLookupErr := handler.accountRepository.FindAccountByProviderAndID(
		ctx,
		googleOAuthProviderName,
		userInfo.Sub,
	)
	if accountLookupErr != nil {
		return nil, accountLookupErr
	}

	if existingAccount != nil && strings.TrimSpace(existingAccount.User.ID) != "" {
		return &existingAccount.User, nil
	}

	normalizedEmail := strings.TrimSpace(userInfo.Email)
	existingUser, userLookupErr := handler.accountRepository.FindUserByEmail(ctx, normalizedEmail)
	if userLookupErr != nil {
		return nil, userLookupErr
	}

	accountID, accountUUIDError := generateRandomUUIDv4()
	if accountUUIDError != nil {
		return nil, accountUUIDError
	}

	if existingUser != nil {
		linkedAccount := &models.Account{
			ID:             accountID,
			UserID:         existingUser.ID,
			Provider:       googleOAuthProviderName,
			ProviderUserID: userInfo.Sub,
			ProviderEmail:  normalizedEmail,
		}

		if linkError := handler.accountRepository.LinkAccountToExistingUser(ctx, linkedAccount); linkError != nil {
			return nil, linkError
		}

		existingUser.Verified = true
		existingUser.EmailVerified = true
		return existingUser, nil
	}

	newUserID, userUUIDError := generateRandomUUIDv4()
	if userUUIDError != nil {
		return nil, userUUIDError
	}

	verifiedNow := time.Now().UTC()
	newUser := &models.User{
		ID:            newUserID,
		Email:         normalizedEmail,
		Password:      "",
		Consented:     false,
		Verified:      true,
		EmailVerified: true,
		VerifiedAt:    &verifiedNow,
	}

	newAccount := &models.Account{
		ID:             accountID,
		UserID:         newUserID,
		Provider:       googleOAuthProviderName,
		ProviderUserID: userInfo.Sub,
		ProviderEmail:  normalizedEmail,
	}

	if createError := handler.accountRepository.CreateOAuthUserAndAccount(ctx, newUser, newAccount); createError != nil {
		return nil, createError
	}

	return newUser, nil
}

// HandleGoogleOAuthCallback handles GET /api/auth/oauth/google/callback:
// validates the OAuth state cookie, exchanges the authorization code for an access token,
// fetches Google's userinfo, resolves/links/creates the User & Account, issues session cookies, and redirects to the dashboard.
func (handler *GoogleOAuthHandler) HandleGoogleOAuthCallback(
	responseWriter http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodGet {
		WriteJSONResponse(responseWriter, http.StatusMethodNotAllowed, MessageOnlyResponsePayload{
			Message: "only GET method is supported",
		})
		return
	}

	queryState := strings.TrimSpace(request.URL.Query().Get("state"))
	cookieState, stateError := auth.ValidateAndClearOAuthStateCookie(responseWriter, request, handler.isProduction)
	if stateError != nil || queryState == "" || subtle.ConstantTimeCompare([]byte(cookieState), []byte(queryState)) != 1 {
		http.Redirect(responseWriter, request, handler.appBaseURL+"/?error=oauth_state_invalid", http.StatusSeeOther)
		return
	}

	authorizationCode := strings.TrimSpace(request.URL.Query().Get("code"))
	if authorizationCode == "" {
		http.Redirect(responseWriter, request, handler.appBaseURL+"/?error=oauth_code_missing", http.StatusSeeOther)
		return
	}

	exchangeCtx := request.Context()
	if handler.httpClient != nil {
		exchangeCtx = context.WithValue(exchangeCtx, oauth2.HTTPClient, handler.httpClient)
	}

	oauthToken, exchangeError := handler.oauthConfig.Exchange(exchangeCtx, authorizationCode)
	if exchangeError != nil {
		http.Redirect(responseWriter, request, handler.appBaseURL+"/?error=oauth_exchange_failed", http.StatusSeeOther)
		return
	}

	userInfo, userInfoError := handler.fetchGoogleUserInfo(request.Context(), oauthToken.AccessToken)
	if userInfoError != nil {
		http.Redirect(responseWriter, request, handler.appBaseURL+"/?error=oauth_userinfo_failed", http.StatusSeeOther)
		return
	}

	userRecord, resolveError := handler.resolveOrCreateFederatedUser(request.Context(), userInfo)
	if resolveError != nil || userRecord == nil {
		http.Redirect(responseWriter, request, handler.appBaseURL+"/?error=oauth_account_error", http.StatusSeeOther)
		return
	}

	rawSecret, secretHash, secretError := auth.GenerateRefreshTokenSecretAndHash()
	sessionID, sessionUUIDError := generateRandomUUIDv4()
	refreshTokenID, tokenUUIDError := generateRandomUUIDv4()
	if secretError != nil || sessionUUIDError != nil || tokenUUIDError != nil {
		http.Redirect(responseWriter, request, handler.appBaseURL+"/?error=oauth_session_error", http.StatusSeeOther)
		return
	}

	expirationTimeUTC := time.Now().UTC().Add(refreshTokenLifetimeDuration)
	var userAgentPointer *string
	if userAgentHeader := strings.TrimSpace(request.UserAgent()); userAgentHeader != "" {
		userAgentPointer = &userAgentHeader
	}

	sessionRecord := &models.Session{
		ID:        sessionID,
		UserID:    userRecord.ID,
		UserAgent: userAgentPointer,
		IPAddress: extractClientIPAddress(request),
		Revoked:   false,
		ExpiresAt: expirationTimeUTC,
	}

	initialRefreshToken := &models.RefreshToken{
		ID:        refreshTokenID,
		SessionID: sessionID,
		TokenHash: secretHash,
		Revoked:   false,
		ExpiresAt: expirationTimeUTC,
	}

	if sessionCreateErr := handler.sessionRepository.CreateSessionWithInitialRefreshToken(request.Context(), sessionRecord, initialRefreshToken); sessionCreateErr != nil {
		http.Redirect(responseWriter, request, handler.appBaseURL+"/?error=oauth_session_error", http.StatusSeeOther)
		return
	}

	signedAccessToken, jwtError := auth.GenerateSignedAccessToken(
		userRecord.ID,
		sessionRecord.ID,
		handler.jwtSecret,
		accessTokenLifetimeDuration,
	)
	if jwtError != nil {
		http.Redirect(responseWriter, request, handler.appBaseURL+"/?error=oauth_session_error", http.StatusSeeOther)
		return
	}

	compositeRefreshToken := auth.FormatCompositeRefreshTokenCookieValue(initialRefreshToken.ID, rawSecret)
	auth.SetAuthenticationCookies(responseWriter, signedAccessToken, compositeRefreshToken, handler.isProduction)

	redirectDestination := handler.appBaseURL + "/dashboard"
	if !userRecord.Consented {
		redirectDestination = handler.appBaseURL + "/dashboard?onboarding=consent"
	}

	http.Redirect(responseWriter, request, redirectDestination, http.StatusSeeOther)
}
