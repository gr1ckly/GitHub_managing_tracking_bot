package proxy

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	internalproxy "coder_manager/internal/proxy"

	"go.uber.org/zap"
)

var ErrSessionNotFound = errors.New("session not found")

type QueryTokenRewriter struct {
	Param string
	Token string
}

func (r QueryTokenRewriter) Rewrite(target *url.URL) error {
	if r.Param == "" {
		return errors.New("query param is empty")
	}
	query := target.Query()
	query.Set(r.Param, r.Token)
	target.RawQuery = query.Encode()
	return nil
}

type ChainRewriter []internalproxy.URLRewriter

func (c ChainRewriter) Rewrite(target *url.URL) error {
	for _, rewriter := range c {
		if rewriter == nil {
			continue
		}
		if err := rewriter.Rewrite(target); err != nil {
			return err
		}
	}
	return nil
}

type Handler struct {
	Store      internalproxy.SessionStore
	Rewriter   internalproxy.URLRewriter
	Clock      func() time.Time
	PathPrefix string
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Store == nil || h.Rewriter == nil {
		http.Error(w, "proxy not configured", http.StatusInternalServerError)
		return
	}
	clock := h.Clock
	if clock == nil {
		clock = time.Now
	}
	prefix := h.PathPrefix
	if prefix == "" {
		prefix = "/edit/"
	}
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	token := strings.TrimPrefix(r.URL.Path, prefix)
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}
	session, err := h.Store.GetSessionByToken(r.Context(), token)
	if err != nil || session == nil {
		zap.S().Warnw("session not found", "error", err)
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if session.ConsumedAt != nil {
		http.Error(w, "session already used", http.StatusGone)
		return
	}
	if session.ExpiresAt != nil && clock().After(*session.ExpiresAt) {
		http.Error(w, "session expired", http.StatusGone)
		return
	}
	target, err := url.Parse(session.SessionURL)
	if err != nil {
		zap.S().Warnw("invalid session url", "error", err)
		http.Error(w, "invalid session url", http.StatusInternalServerError)
		return
	}
	if err := h.Rewriter.Rewrite(target); err != nil {
		zap.S().Warnw("rewrite failed", "error", err)
		http.Error(w, "invalid redirect", http.StatusInternalServerError)
		return
	}
	_ = h.Store.MarkSessionConsumed(context.Background(), session.ID, clock())
	http.Redirect(w, r, target.String(), http.StatusTemporaryRedirect)
}
