package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	guestCookieName      = "iss_guest"
	guestSessionTTL      = 20 * time.Minute
	guestSessionKeyPref  = "guest:sess:"
	guestViewedKeyPrefix = "guest:rv:"
	guestViewedMax       = 50
)

func ensureGuestSession(w http.ResponseWriter, r *http.Request) string {
	now := time.Now()
	guestID := readGuestCookie(r)
	if guestID == "" {
		return createGuestSession(w, now)
	}

	createdAt, ok := getGuestSessionCreatedAt(r.Context(), guestID)
	if !ok {
		return createGuestSession(w, now)
	}

	age := now.Sub(createdAt)
	if age < 0 || age >= guestSessionTTL {
		return createGuestSession(w, now)
	}

	remaining := guestSessionTTL - age
	setGuestCookie(w, guestID, remaining)
	refreshGuestSessionTTL(r.Context(), guestID, remaining)
	return guestID
}

func recordViewedService(ctx context.Context, guestID string, serviceID int) {
	if guestID == "" || serviceID <= 0 {
		return
	}
	rc, err := getRedisClient()
	if err != nil {
		log.Printf("guest redis unavailable (record view): %v", err)
		return
	}
	createdAt, ok := getGuestSessionCreatedAt(ctx, guestID)
	if !ok {
		return
	}
	remaining := guestSessionTTL - time.Since(createdAt)
	if remaining <= 0 {
		return
	}

	key := guestViewedKeyPrefix + guestID
	idStr := strconv.Itoa(serviceID)
	if err := rc.LRem(ctx, key, 0, idStr).Err(); err != nil {
		log.Printf("guest viewed lrem error: %v", err)
		return
	}
	if err := rc.LPush(ctx, key, idStr).Err(); err != nil {
		log.Printf("guest viewed lpush error: %v", err)
		return
	}
	if err := rc.LTrim(ctx, key, 0, guestViewedMax-1).Err(); err != nil {
		log.Printf("guest viewed ltrim error: %v", err)
		return
	}
	if err := rc.Expire(ctx, key, remaining).Err(); err != nil {
		log.Printf("guest viewed expire error: %v", err)
	}
}

func getRecentlyViewedServiceIDs(ctx context.Context, guestID string, limit int) []int {
	if guestID == "" {
		return nil
	}
	if limit <= 0 {
		limit = 6
	}
	if limit > guestViewedMax {
		limit = guestViewedMax
	}

	rc, err := getRedisClient()
	if err != nil {
		log.Printf("guest redis unavailable (load viewed): %v", err)
		return nil
	}
	createdAt, ok := getGuestSessionCreatedAt(ctx, guestID)
	if !ok {
		return nil
	}
	remaining := guestSessionTTL - time.Since(createdAt)
	if remaining <= 0 {
		return nil
	}
	refreshGuestSessionTTL(ctx, guestID, remaining)
	_ = rc.Expire(ctx, guestViewedKeyPrefix+guestID, remaining).Err()

	raw, err := rc.LRange(ctx, guestViewedKeyPrefix+guestID, 0, int64(limit-1)).Result()
	if err != nil {
		log.Printf("guest viewed lrange error: %v", err)
		return nil
	}

	ids := make([]int, 0, len(raw))
	seen := make(map[int]struct{}, len(raw))
	for _, item := range raw {
		id, convErr := strconv.Atoi(item)
		if convErr != nil || id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func createGuestSession(w http.ResponseWriter, now time.Time) string {
	guestID := newGuestID()
	setGuestCookie(w, guestID, guestSessionTTL)
	if rc, err := getRedisClient(); err == nil {
		key := guestSessionKeyPref + guestID
		_ = rc.Set(context.Background(), key, strconv.FormatInt(now.Unix(), 10), guestSessionTTL).Err()
	} else {
		log.Printf("guest redis unavailable (create session): %v", err)
	}
	return guestID
}

func getGuestSessionCreatedAt(ctx context.Context, guestID string) (time.Time, bool) {
	if guestID == "" {
		return time.Time{}, false
	}
	rc, err := getRedisClient()
	if err != nil {
		return time.Time{}, false
	}
	value, err := rc.Get(ctx, guestSessionKeyPref+guestID).Result()
	if err != nil {
		return time.Time{}, false
	}
	sec, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(sec, 0), true
}

func refreshGuestSessionTTL(ctx context.Context, guestID string, ttl time.Duration) {
	if guestID == "" || ttl <= 0 {
		return
	}
	rc, err := getRedisClient()
	if err != nil {
		return
	}
	_ = rc.Expire(ctx, guestSessionKeyPref+guestID, ttl).Err()
}

func readGuestCookie(r *http.Request) string {
	c, err := r.Cookie(guestCookieName)
	if err != nil {
		return ""
	}
	v := strings.TrimSpace(c.Value)
	if v == "" {
		return ""
	}
	return v
}

func setGuestCookie(w http.ResponseWriter, guestID string, ttl time.Duration) {
	if ttl <= 0 {
		ttl = guestSessionTTL
	}
	http.SetCookie(w, &http.Cookie{
		Name:     guestCookieName,
		Value:    guestID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
	})
}

func newGuestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(b)
}
