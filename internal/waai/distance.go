package waai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type geoPoint struct {
	Lat float64
	Lon float64
}

func estimateTripDistanceKm(ctx context.Context, from, to string) (float64, string, error) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" {
		return 0, "", fmt.Errorf("from and to are required")
	}

	fromPt, err := geocodeIndonesia(ctx, from)
	if err != nil {
		return 0, "", fmt.Errorf("failed to geocode origin: %w", err)
	}
	toPt, err := geocodeIndonesia(ctx, to)
	if err != nil {
		return 0, "", fmt.Errorf("failed to geocode destination: %w", err)
	}

	if km, err := osrmDistanceKm(ctx, fromPt, toPt); err == nil {
		return km, "osrm", nil
	}

	km := haversineKm(fromPt, toPt) * 1.2
	return km, "haversine", nil
}

func geocodeIndonesia(ctx context.Context, query string) (geoPoint, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return geoPoint{}, fmt.Errorf("query is empty")
	}
	if !strings.Contains(strings.ToLower(q), "indonesia") {
		q = q + ", Indonesia"
	}

	endpoint, _ := url.Parse("https://nominatim.openstreetmap.org/search")
	values := endpoint.Query()
	values.Set("format", "json")
	values.Set("limit", "1")
	values.Set("countrycodes", "id")
	values.Set("q", q)
	endpoint.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return geoPoint{}, err
	}
	req.Header.Set("User-Agent", "service-travego/waai")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return geoPoint{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return geoPoint{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return geoPoint{}, fmt.Errorf("geocode status %d", resp.StatusCode)
	}

	var rows []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return geoPoint{}, err
	}
	if len(rows) == 0 {
		return geoPoint{}, fmt.Errorf("not found")
	}

	lat, err := strconv.ParseFloat(strings.TrimSpace(rows[0].Lat), 64)
	if err != nil {
		return geoPoint{}, err
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(rows[0].Lon), 64)
	if err != nil {
		return geoPoint{}, err
	}

	return geoPoint{Lat: lat, Lon: lon}, nil
}

func osrmDistanceKm(ctx context.Context, from, to geoPoint) (float64, error) {
	urlStr := fmt.Sprintf(
		"https://router.project-osrm.org/route/v1/driving/%f,%f;%f,%f?overview=false",
		from.Lon, from.Lat, to.Lon, to.Lat,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "service-travego/waai")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("osrm status %d", resp.StatusCode)
	}

	var parsed struct {
		Routes []struct {
			Distance float64 `json:"distance"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, err
	}
	if len(parsed.Routes) == 0 {
		return 0, fmt.Errorf("no routes")
	}
	if parsed.Routes[0].Distance <= 0 {
		return 0, fmt.Errorf("invalid distance")
	}
	return parsed.Routes[0].Distance / 1000.0, nil
}

func haversineKm(a, b geoPoint) float64 {
	const earthRadiusKm = 6371.0
	lat1 := degreesToRadians(a.Lat)
	lon1 := degreesToRadians(a.Lon)
	lat2 := degreesToRadians(b.Lat)
	lon2 := degreesToRadians(b.Lon)

	dLat := lat2 - lat1
	dLon := lon2 - lon1
	sinLat := math.Sin(dLat / 2)
	sinLon := math.Sin(dLon / 2)

	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(h))
}

func degreesToRadians(deg float64) float64 {
	return deg * math.Pi / 180.0
}

