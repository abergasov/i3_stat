package sampler

import (
	"compress/gzip"
	"container/list"
	"encoding/json"
	"fmt"
	"i3_stat/internal/logger"
	"i3_stat/internal/models"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	baseDomain = "https://min-api.cryptocompare.com/data/price"
)

type Service struct {
	compareAPIKeys   *list.List
	compareAPIKeysMU sync.Mutex
	log              logger.AppLogger
	priceMU          sync.RWMutex
	observePrice     []models.Coin
	observeList      []string
}

func InitService(log logger.AppLogger, compareAPIKey []string) *Service {
	apiList := list.New()
	for _, key := range compareAPIKey {
		apiList.PushBack(key)
	}
	srv := &Service{
		compareAPIKeys: apiList,
		log:            log.With(zap.String("service", "sampler")),
		observeList: []string{
			models.Bitcoin,
			models.Ethereum,
			//models.Atom,
			//models.Polkadot,
			models.Chia,
		},
	}
	for _, tk := range srv.observeList {
		srv.observePrice = append(srv.observePrice, models.Coin{Ticker: tk})
	}
	go srv.observePrices()
	return srv
}

func (s *Service) observePrices() {
	go s.getPrices()
	ticker := time.NewTicker(15 * time.Second)
	for range ticker.C {
		go s.getPrices()
	}
}

func (s *Service) getPrices() {
	for i := range s.observeList {
		go func(index int, ticker string) {
			assetPrice, err := s.loadPrice(ticker)
			if err != nil {
				s.log.Error("failed to load price", err, zap.String("ticker", ticker))
				return
			}
			s.priceMU.Lock()
			s.observePrice[index].Price = assetPrice
			s.priceMU.Unlock()
		}(i, s.observeList[i])
	}
}

func (s *Service) getAPIKey() string {
	s.compareAPIKeysMU.Lock()
	defer s.compareAPIKeysMU.Unlock()
	key := s.compareAPIKeys.Front()
	s.compareAPIKeys.MoveToBack(key)
	return key.Value.(string)
}

func (s *Service) loadPrice(targetCurrency string) (float64, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?fsym=%s&tsyms=USD", baseDomain, targetCurrency), http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	apiKey := s.getAPIKey()
	req.Header.Set("authorization", "Apikey "+apiKey)
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
		s.log.Error("failed to unmarshal response body", err, zap.Int("code", resp.StatusCode), zap.String("api", apiKey))
		return 0, fmt.Errorf("failed to unmarshal response body: %w", err)
	}
	return res["USD"], nil
}

func (s *Service) GetState() string {
	s.priceMU.RLock()
	defer s.priceMU.RUnlock()
	stringsList := make([]string, 0, len(s.observeList))
	for _, t := range s.observePrice {
		stringsList = append(stringsList, fmt.Sprintf("%s: %.2f", t.Ticker, t.Price))
	}
	return strings.Join(stringsList, ", ")
}
