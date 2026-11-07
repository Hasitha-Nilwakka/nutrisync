// This file implements HTTP handlers for GitHub OAuth 2.0 login initiation (GET /api/auth/oauth/github)
// and callback handling with primary verified email selection and federated account linking (GET /api/auth/oauth/github/callback).
package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"nutrysync/go-api/auth"
	"nutrysync/go-api/models"
	"nutrysync/go-api/repositories"
)

const (
	githubOAuthProviderName         = "github"
	defaultGitHubUserEndpoint       = "https://api.github.com/user"
	defaultGitHubUserEmailsEndpoint = "https://api.github.com/user/emails"
)

// GitHubUserProfilePayload represents the JSON response from https://api.github.com/user.
type GitHubUserProfilePayload struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
}

// GitHubUserEmailEntry represents a single email record from https://api.github.com/user/emails.
type GitHubUserEmailEntry struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// GitHubOAuthHandler coordinates GitHub OAuth 2.0 login redirects, callback verification,
// verified email resolution, federated account linking, and session issuance.
type GitHubOAuthHandler struct {
	accountRepository     repositories.AccountRepository
	sessionRepository     repositories.UserSessionRepository
	oauthConfig           *oauth2.Config
	jwtSecret             string
	appBaseURL            string
	isProduction          bool
	userEndpointURL       string
	userEmailsEndpointURL string
	httpClient            *http.Client
}

// NewGitHubOAuthHandler constructs a GitHubOAuthHandler after validating required dependencies.
func NewGitHubOAuthHandler(
	accountRepository repositories.AccountRepository,
	sessionRepository repositories.UserSessionRepository,
	oauthConfig *oauth2.Config,
	jwtSecret string,
	appBaseURL string,
	isProduction bool,
) (*GitHubOAuthHandler, error) {
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

	return &GitHubOAuthHandler{
		accountRepository:     accountRepository,
		sessionRepository:     sessionRepository,
		oauthConfig:           oauthConfig,
		jwtSecret:             trimmedJWTSecret,
		appBaseURL:            normalizedBaseURL,
		isProduction:          isProduction,
		userEndpointURL:       defaultGitHubUserEndpoint,
		userEmailsEndpointURL: defaultGitHubUserEmailsEndpoint,
		httpClient:            http.DefaultClient,
	}, nil
}

// HandleGitHubOAuthLogin handles GET /api/auth/oauth/github:
// generates a random CSRF state token, stores it in an HttpOnly cookie, and redirects (307) to GitHub's authorization URL.
func (handler *GitHubOAuthHandler) HandleGitHubOAuthLogin(
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
	consentRedirectURL := handler.oauthConfig.AuthCodeURL(stateToken)
	http.Redirect(responseWriter, request, consentRedirectURL, http.StatusTemporaryRedirect)
}

// SelectVerifiedGitHubEmail selects the primary verified email from a slice of GitHubUserEmailEntry records,
// falling back to the first verified email if none is marked primary.
func SelectVerifiedGitHubEmail(emailEntries []GitHubUserEmailEntry) (string, bool) {
	for _, entry := range emailEntries {
		if entry.Primary && entry.Verified && strings.TrimSpace(entry.Email) != "" {
			return strings.TrimSpace(entry.Email), true
		}
	}

	for _, entry := range emailEntries {
		if entry.Verified && strings.TrimSpace(entry.Email) != "" {
			return strings.TrimSpace(entry.Email), true
		}
	}

	return "", false
}

// fetchGitHubUserID retrieves the authenticated GitHub user's numerical ID formatted as a string.
func (handler *GitHubOAuthHandler) fetchGitHubUserID(
	ctx context.Context,
	accessToken string,
) (string, error) {
	endpointURL := strings.TrimSpace(handler.userEndpointURL)
	if endpointURL == "" {
		endpointURL = defaultGitHubUserEndpoint
	}

	userRequest, requestError := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
	if requestError != nil {
		return "", requestError
	}
	userRequest.Header.Set("Authorization", "Bearer "+accessToken)
	userRequest.Header.Set("Accept", "application/vnd.github+json")

	httpClient := handler.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	response, responseError := httpClient.Do(userRequest)
	if responseError != nil {
		return "", responseError
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github user endpoint returned status %d", response.StatusCode)
	}

	var profile GitHubUserProfilePayload
	if decodeError := json.NewDecoder(response.Body).Decode(&profile); decodeError != nil {
		return "", decodeError
	}

	if profile.ID <= 0 {
		return "", errors.New("github user ID is missing or invalid")
	}

	return strconv.FormatInt(profile.ID, 10), nil
}

// fetchGitHubVerifiedEmail retrieves the authenticated GitHub user's email list and selects a verified email address.
func (handler *GitHubOAuthHandler) fetchGitHubVerifiedEmail(
	ctx context.Context,
	accessToken string,
) (string, bool, error) {
	endpointURL := strings.TrimSpace(handler.userEmailsEndpointURL)
	if endpointURL == "" {
		endpointURL = defaultGitHubUserEmailsEndpoint
	}

	emailsRequest, requestError := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
	if requestError != nil {
		return "", false, requestError
	}
	emailsRequest.Header.Set("Authorization", "Bearer "+accessToken)
	emailsRequest.Header.Set("Accept", "application/vnd.github+json")

	httpClient := handler.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	response, responseError := httpClient.Do(emailsRequest)
	if responseError != nil {
		return "", false, responseError
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("github emails endpoint returned status %d", response.StatusCode)
	}

	var emailEntries []GitHubUserEmailEntry
	if decodeError := json.NewDecoder(response.Body).Decode(&emailEntries); decodeError != nil {
		return "", false, decodeError
	}

	verifiedEmail, found := SelectVerifiedGitHubEmail(emailEntries)
	return verifiedEmail, found, nil
}

// resolveOrCreateGitHubUser resolves an existing User by GitHub Account or Email, or creates a new User + GitHub Account.
func (handler *GitHubOAuthHandler) resolveOrCreateGitHubUser(
	ctx context.Context,
	githubUserID string,
	verifiedEmail string,
) (*models.User, error) {
	existingAccount, accountLookupErr := handler.accountRepository.FindAccountByProviderAndID(
		ctx,
		githubOAuthProviderName,
		githubUserID,
	)
	if accountLookupErr != nil {
		return nil, accountLookupErr
	}

	if existingAccount != nil && strings.TrimSpace(existingAccount.User.ID) != "" {
		return &existingAccount.User, nil
	}

	normalizedEmail := strings.TrimSpace(verifiedEmail)
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
			Provider:       githubOAuthProviderName,
			ProviderUserID: githubUserID,
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
		Provider:       githubOAuthProviderName,
		ProviderUserID: githubUserID,
		ProviderEmail:  normalizedEmail,
	}

	if createError := handler.accountRepository.CreateOAuthUserAndAccount(ctx, newUser, newAccount); createError != nil {
		return nil, createError
	}

	return newUser, nil
}

// HandleGitHubOAuthCallback handles GET /api/auth/oauth/github/callback:
// validates the OAuth state cookie, exchanges code for token, fetches GitHub user ID and verified primary email,
// resolves/links/creates the User & Account, issues session cookies, and redirects to the dashboard.
func (handler *GitHubOAuthHandler) HandleGitHubOAuthCallback(
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

	githubUserID, userIDError := handler.fetchGitHubUserID(request.Context(), oauthToken.AccessToken)
	if userIDError != nil {
		http.Redirect(responseWriter, request, handler.appBaseURL+"/?error=oauth_userinfo_failed", http.StatusSeeOther)
		return
	}

	verifiedEmail, foundVerifiedEmail, emailsError := handler.fetchGitHubVerifiedEmail(request.Context(), oauthToken.AccessToken)
	if emailsError != nil {
		http.Redirect(responseWriter, request, handler.appBaseURL+"/?error=oauth_userinfo_failed", http.StatusSeeOther)
		return
	}

	if !foundVerifiedEmail || verifiedEmail == "" {
		http.Redirect(responseWriter, request, handler.appBaseURL+"/?error=oauth_email_unverified", http.StatusSeeOther)
		return
	}

	userRecord, resolveError := handler.resolveOrCreateGitHubUser(request.Context(), githubUserID, verifiedEmail)
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
