package sampler

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"i3_stat/internal/logger"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	baseDomain = "https://min-api.cryptocompare.com/data/price"
)

type Service struct {
	compareAPIKey string
	log           logger.AppLogger
	priceMU       *sync.RWMutex
	btcPrice      float64
	ethPrice      float64
}

func InitService(log logger.AppLogger, compareAPIKey string) *Service {
	srv := &Service{
		compareAPIKey: compareAPIKey,
		log:           log.With(zap.String("service", "sampler")),
		priceMU:       &sync.RWMutex{},
	}
	go srv.observePrices()
	return srv
}

func (s *Service) observePrices() {
	go s.getBTCPrice()
	go s.getETHPrice()
	ticker := time.NewTicker(15 * time.Second)
	for range ticker.C {
		s.getBTCPrice()
		s.getETHPrice()
	}
}

func (s *Service) getBTCPrice() {
	btcPrice, err := s.loadPrice("BTC")
	if err != nil {
		s.log.Error("failed to load BTC price", err)
		return
	}
	s.priceMU.Lock()
	s.btcPrice = btcPrice
	s.priceMU.Unlock()
}

func (s *Service) getETHPrice() {
	ethPrice, err := s.loadPrice("ETH")
	if err != nil {
		s.log.Error("failed to load ETH price", err)
		return
	}
	s.priceMU.Lock()
	s.ethPrice = ethPrice
	s.priceMU.Unlock()
}

func (s *Service) loadPrice(targetCurrency string) (float64, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?fsym=%s&tsyms=USD", baseDomain, targetCurrency), http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("authorization", "Apikey "+s.compareAPIKey)
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to load price: %w", err)
	}
	defer resp.Body.Close()
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return 0, fmt.Errorf("failed to create gzip reader: %w", err)
		}
	default:
		reader = resp.Body
	}
	defer reader.Close()
	b, err := io.ReadAll(reader)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}
	var res map[string]float64
	if err = json.Unmarshal(b, &res); err != nil {
		return 0, fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return res["USD"], nil
}

func (s *Service) GetState() string {
	s.priceMU.RLock()
	defer s.priceMU.RUnlock()
	return fmt.Sprintf("BTC: %.2f, ETH: %.2f", s.btcPrice, s.ethPrice)
}
